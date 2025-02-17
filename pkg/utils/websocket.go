package utils

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type GroupWebSocketsState struct {
	sockets     map[string][]chan []byte // username -> multiple channels
	mu          sync.RWMutex
	authHandler GooglePublicKeysHandler
	db          *sql.DB
}

func (self *GroupWebSocketsState) AddGroupMessage(groupMsg *GroupChannelMessage, verifiedToken *VerifiedToken) {

	if verifiedToken == nil || 0 == len(verifiedToken.Username) {
		log.Printf("ERROR: User is not authenticated")
		return
	}
	groupMsg.By = verifiedToken.Username
	groupMsg, err := AddChannelMessage(self.db, groupMsg)
	if err != nil {
		log.Printf("ERROR: Adding channel message %s", err)
		return
	}

	users := GetUsersInChannel(self.db, int(groupMsg.GetInChannel()))
	finalMsg := GroupMessage{
		Message: &GroupMessage_GroupMessage{
			GroupMessage: groupMsg,
		},
	}

	payload, err := proto.Marshal(&finalMsg)
	if err != nil {
		log.Printf("ERROR: Encoding protobuf %s", err)
	}

	{
		self.mu.RLock()
		defer self.mu.RUnlock()
		for _, user := range users {
			go self.SendMessageToUser(user.username, payload)
		}
	}
}

func NewGroupWebSocketsState() GroupWebSocketsState {

	audience := os.Getenv("AUDIENCE")

	if len(audience) == 0 {
		log.Printf("WARN: AUDIENCE environment variable is empty")
	}

	databaseFile := os.Getenv("DATABASE_FILE")
	if len(databaseFile) == 0 {
		log.Fatal("WARN: DATABASE_FILE environment variable is empty")
	}

	creator := os.Getenv("CREATOR")
	if len(creator) == 0 {
		log.Fatal("WARN: CREATOR environment variable is empty")
	}
	return GroupWebSocketsState{
		sockets:     make(map[string][]chan []byte),
		authHandler: NewGooglePublicKeysHandler(audience),
		db:          OpenDatabase(databaseFile, creator),
	}
}

// WARN: Is it reliable to compare channels for equality directly ?

func (self *GroupWebSocketsState) NewSocket(username string, writer chan []byte) {
	self.mu.Lock()
	defer self.mu.Unlock()

	writers, found := self.sockets[username]
	if found {

		for _, existingWriter := range writers {
			if existingWriter == writer {
				return
			}
		}

		writers = append(writers, writer)
	} else {
		writers := make([]chan []byte, 0, 1)
		writers = append(writers, writer)
		self.sockets[username] = writers
	}
}

func (self *GroupWebSocketsState) SocketClosed(username string, writer chan []byte) {
	self.mu.Lock()
	defer self.mu.Unlock()

	writers, found := self.sockets[username]
	if found {
		for idx, existingWriter := range writers {
			if existingWriter == writer {
				writers[idx] = writers[len(writers)-1]
				writers = writers[:len(writers)-1]

				if len(writers) == 0 {
					delete(self.sockets, username)
				}
				break
			}
		}
	}
}

func (self *GroupWebSocketsState) SendMessageToUsers(users []string, payload []byte) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	for _, user := range users {
		go self.SendMessageToUser(user, payload)
	}
}

func (self *GroupWebSocketsState) SendMessageToUser(username string, payload []byte) {
	var wg sync.WaitGroup
	sockets, ok := self.sockets[username]
	if ok {
		for _, socket := range sockets {
			wg.Add(1)
			go func() {
				defer wg.Done()
				socket <- payload
			}()
		}
	}

	wg.Wait()
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: true,
}

func WebsocketHandler(w http.ResponseWriter, r *http.Request, server *GroupWebSocketsState) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	go handleWebSocket(conn, server)
}

func handleWebSocket(conn *websocket.Conn, state *GroupWebSocketsState) {

	messageSink := make(chan []byte)
	var verifiedToken *VerifiedToken

	defer func() {
		conn.Close()
		if verifiedToken != nil {
			state.SocketClosed(verifiedToken.Username, messageSink)
		}
	}()

	go func() {
		for payload := range messageSink {
			if err := conn.WriteMessage(websocket.BinaryMessage, payload); err != nil {
				log.Printf("ERROR: writing message, %s", err)
				break
			}
		}
	}()

	for {
		msgType, msg, err := conn.ReadMessage()
		_ = msgType
		if err != nil {
			log.Println(err)
			break
		}
		message := GroupMessage{}
		err = proto.Unmarshal(msg, &message)
		if err != nil {
			log.Printf("received invalid message %s", err)
		}

		switch x := message.Message.(type) {
		case *GroupMessage_AuthToken:
			{
				token := x.AuthToken.Token
				verifiedToken = state.authHandler.VerifyToken(token)
				if verifiedToken != nil {
					state.NewSocket(verifiedToken.Username, messageSink)
				}
			}
		case *GroupMessage_GroupMessage:
			{
				groupMsg := x.GroupMessage
				go state.AddGroupMessage(groupMsg, verifiedToken)
			}

		default:
		}
	}
}

func StartServer(addr string) {
	state := NewGroupWebSocketsState()
	fs := http.FileServer(http.Dir("./public"))

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { WebsocketHandler(w, r, &state) })
	http.Handle("/", http.StripPrefix("/", fs))
	http.HandleFunc("/egg", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Egg"))
	})

	log.Println("Serving at ", addr)
	http.ListenAndServe(addr, nil)
}
