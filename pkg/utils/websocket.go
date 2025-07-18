package utils

import (
	"firefly/pkg/protos"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

type GroupWebSocketsState struct {
	sockets     map[string][]chan []byte // username -> multiple channels
	mu          sync.RWMutex             // mutex for the above map
	authHandler PublicKeysHandler
	db          *pgxpool.Pool
}

type WebSocketState struct {
	sockets map[string][]chan []byte
	mu      sync.RWMutex
}

func (self *GroupWebSocketsState) RemoveUser(groupId int, removeUser *protos.RemoveUser, verifiedToken *VerifiedToken) {
	if verifiedToken == nil || 0 == len(verifiedToken.Username) {
		log.Printf("ERROR: User is not authenticated")
		return
	}

	username := removeUser.GetUsername()
	channel := int(removeUser.GetFromChannel())

	RemoveUserFromChannel(self.db, username, groupId, channel, verifiedToken.Username)
}

func (self *GroupWebSocketsState) AddUser(groupId int, addUser *protos.AddUser, verifiedToken *VerifiedToken) {

	if verifiedToken == nil || 0 == len(verifiedToken.Username) {
		log.Printf("ERROR: User is not authenticated")
		return
	}

	username := addUser.GetUsername()
	channel := int(addUser.GetToChannel())
	role := int(addUser.GetRole())

	AddUserToChannel(self.db, username, groupId, channel, role, verifiedToken.Username)

}

func (self *GroupWebSocketsState) HandleGroupMessage(msg *protos.GroupMessage, verifiedToken *VerifiedToken) {

	groupId := int(msg.GetGroupId())

	switch x := msg.Message.(type) {

	case *protos.GroupMessage_AddChannel:
		message := x.AddChannel
		CreateChannel(self.db, groupId, message.GetName(), int(message.GetChannelType()), verifiedToken.Username)
	case *protos.GroupMessage_AddUser:
		message := x.AddUser
		AddUserToChannel(self.db, message.GetUsername(), groupId, int(message.GetToChannel()), int(message.GetRole()), verifiedToken.Username)

	case *protos.GroupMessage_DeleteChannel:

		DeleteChannel(self.db, groupId, int(x.DeleteChannel.GetId()))

	case *protos.GroupMessage_GroupMessage:
		self.AddGroupMessage(groupId, x.GroupMessage, verifiedToken)
	case *protos.GroupMessage_RemoveUser:
		RemoveUserFromChannel(self.db, x.RemoveUser.GetUsername(), groupId, int(x.RemoveUser.GetFromChannel()), verifiedToken.Username)
	default:
	}

}

func (self *GroupWebSocketsState) HandleRequest(msg *protos.Request, verifiedToken *VerifiedToken) *protos.Response {
	return nil
}

func (self *GroupWebSocketsState) AddGroupMessage(groupId int, groupMsg *protos.GroupChannelMessage, verifiedToken *VerifiedToken) error {

	if verifiedToken == nil || 0 == len(verifiedToken.Username) {
		return fmt.Errorf("ERROR: User is not authenticated")
	}
	groupMsg.By = verifiedToken.Username
	groupMsg, err := AddChannelMessage(self.db, groupId, groupMsg)
	if err != nil {
		return err
	}

	users, err := GetUsersInChannel(self.db, groupId, int(groupMsg.GetInChannel()))

	if err != nil {
		return err
	}
	finalMsg := protos.GroupMessage{
		Message: &protos.GroupMessage_GroupMessage{
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
		var wg sync.WaitGroup
		for _, user := range users {
			self.SendMessageToUser(user.GetUsername(), payload, &wg)
		}

		go wg.Wait()
	}

	return nil
}

func NewGroupWebSocketsState() GroupWebSocketsState {

	audience := os.Getenv("AUDIENCE")

	if len(audience) == 0 {
		log.Printf("WARN: AUDIENCE environment variable is empty")
	}

	dbUrl := os.Getenv("DB_URL")
	if len(dbUrl) == 0 {
		log.Fatal("DB_URL env var is not set")
	}

	db, err := OpenDatabase(dbUrl)
	if err != nil {
		log.Fatal(err)
	}

	return GroupWebSocketsState{
		sockets:     make(map[string][]chan []byte),
		authHandler: NewPublicKeysHandler(audience),
		db:          db,
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
		writers = []chan []byte{writer}
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
	var wg sync.WaitGroup
	for _, user := range users {
		self.SendMessageToUser(user, payload, &wg)
	}

	go wg.Wait()
}

func (self *GroupWebSocketsState) SendMessageToUser(username string, payload []byte, wg *sync.WaitGroup) {
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

		{
			message := protos.ClientMessage{}
			err := proto.Unmarshal(msg, &message)
			if err != nil {
				log.Printf("received invalid message %s", err)
			}

			if verifiedToken == nil {

				switch x := message.Payload.(type) {

				case *protos.ClientMessage_AuthToken:
					{
						token := x.AuthToken.GetToken()
						verifiedToken = state.authHandler.VerifyToken(token)
						if verifiedToken != nil {
							state.NewSocket(verifiedToken.Username, messageSink)
						}
					}
				default:
					continue
				}

			}

			switch x := message.Payload.(type) {
			case *protos.ClientMessage_GroupMessage:
				{
					groupMsg := x.GroupMessage
					state.HandleGroupMessage(groupMsg, verifiedToken)
				}
			case *protos.ClientMessage_Request:
				{
					request := x.Request
					response := state.HandleRequest(request, verifiedToken)

					if response != nil {
						body, err := proto.Marshal(&protos.ServerMessage{
							Message: &protos.ServerMessage_Response{
								Response: response,
							},
						})
						if err != nil {
							log.Println("Failed to encode Response")
							continue
						}

						messageSink <- body
					}
				}
			default:
			}
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
