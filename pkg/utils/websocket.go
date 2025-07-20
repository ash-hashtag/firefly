package utils

import (
	"firefly/pkg/protos"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

type GroupChannelIdentifier struct {
	groupId, channelId int
}

type GroupWebSocketsState struct {
	authHandler      PublicKeysHandler
	db               *pgxpool.Pool
	groupChannelSubs PubSub[GroupChannelIdentifier]
	activeSockets    PubSub[string]
}

type Subscriber = chan []byte

type PubSub[T comparable] struct {
	subscribers map[T]map[chan []byte]bool
	mu          sync.RWMutex
}

func (self *PubSub[T]) Subscribe(topic T, subscriber Subscriber) {
	self.mu.Lock()
	defer self.mu.Unlock()
	var subscribers, ok = self.subscribers[topic]
	if !ok {
		subscribers = make(map[chan []byte]bool, 1)
	}
	subscribers[subscriber] = true
	self.subscribers[topic] = subscribers

}

func (self *PubSub[T]) Unsubscribe(topic T, subscriber Subscriber) {
	self.mu.Lock()
	defer self.mu.Unlock()
	subscribers, ok := self.subscribers[topic]
	if !ok {
		log.Printf("WARN the topic doesn't exist grp: %v", topic)
		return
	}
	delete(subscribers, subscriber)
}

func (self *PubSub[T]) SendTo(topic T, payload []byte) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	subscribers, ok := self.subscribers[topic]
	if !ok {
		return
	}

	for subscriber := range subscribers {
		go func(s chan []byte) {
			s <- payload
		}(subscriber)
	}
}

func removeElementAtUnordered[T any](arr []T, index int) []T {
	length := len(arr)
	if length == 0 {
		log.Fatal("Can't remove from an empty array")
	}
	arr[index] = arr[length-1]
	return arr[0 : length-1]
}

func (self *GroupWebSocketsState) RemoveUser(removeUser *protos.RemoveUser, verifiedToken *VerifiedToken) {
	if verifiedToken == nil || 0 == len(verifiedToken.Username) {
		log.Printf("ERROR: User is not authenticated")
		return
	}

	username := removeUser.GetUsername()
	channel := int(removeUser.GetGroupId())
	groupId := int(removeUser.GetGroupId())

	RemoveUserFromChannel(self.db, username, groupId, channel, verifiedToken.Username)
}

func (self *GroupWebSocketsState) AddUser(groupId int, addUser *protos.AddUser, verifiedToken *VerifiedToken) {

	if verifiedToken == nil || 0 == len(verifiedToken.Username) {
		log.Printf("ERROR: User is not authenticated")
		return
	}

	username := addUser.GetUsername()
	channel := int(addUser.GetChannelId())
	role := int(addUser.GetRole())

	AddUserToChannel(self.db, username, groupId, channel, role, verifiedToken.Username)

}

func (self *GroupWebSocketsState) HandleRequest(msg *protos.Request, verifiedToken *VerifiedToken) *protos.Response {

	switch x := msg.Message.(type) {

	case *protos.Request_GetGroupMembers:
		users, err := GetUsersInChannel(self.db, int(x.GetGroupMembers.GetGroupId()), int(x.GetGroupMembers.GetChannelId()))
		if err != nil {
			log.Print(err)
			return nil
		}

		self.activeSockets.mu.RLock()
		defer self.activeSockets.mu.RUnlock()
		for _, user := range users {
			_, isOnline := self.activeSockets.subscribers[user.GetUsername()]
			user.IsOnline = isOnline
		}

		return &protos.Response{Message: &protos.Response_Members{Members: &protos.GroupMembers{Members: users}}}

	case *protos.Request_GetGroupMessages:

		before := x.GetGroupMessages.GetBefore()
		if len(before) != 16 {
			return &protos.Response{Message: &protos.Response_Error{Error: &protos.Error{Error: "invalid fields"}}}
		}
		var count = x.GetGroupMessages.GetCount()
		if count > 100 {
			count = 100
		}

		messages, err := GetMessages(self.db, int(x.GetGroupMessages.GetGroupId()), int(x.GetGroupMessages.GetChannelId()), uuid.UUID([16]byte(x.GetGroupMessages.GetBefore())), uint64(count))
		if err != nil {
			log.Print(err)
			return &protos.Response{Message: &protos.Response_Error{Error: &protos.Error{Error: "Internal error"}}}
		}

		return &protos.Response{Message: &protos.Response_Messages{Messages: &protos.GroupChannelMessages{Messages: messages}}}

	case *protos.Request_AddChannel:
	case *protos.Request_AddUser:
	case *protos.Request_DeleteChannel:
	case *protos.Request_RemoveUser:
	default:
		panic(fmt.Sprintf("unexpected protos.isRequest_Message: %#v", x))
	}

	return nil
}

func (self *GroupWebSocketsState) AddGroupMessage(groupMsg *protos.GroupChannelMessage, verifiedToken *VerifiedToken) error {

	if verifiedToken == nil || 0 == len(verifiedToken.Username) {
		return fmt.Errorf("ERROR: User is not authenticated")
	}
	groupMsg.By = verifiedToken.Username
	groupId := int(groupMsg.GetGroupId())
	channelId := int(groupMsg.ChannelId)
	groupMsg, err := AddChannelMessage(self.db, groupMsg)
	if err != nil {
		return err
	}
	finalMsg := groupMsg
	payload, err := proto.Marshal(&protos.ServerMessage{Message: &protos.ServerMessage_GroupMessage{GroupMessage: finalMsg}})
	if err != nil {
		log.Printf("ERROR: Encoding protobuf %s", err)
	}

	{
		self.groupChannelSubs.mu.RLock()
		defer self.groupChannelSubs.mu.RUnlock()
		subscribers, ok := self.groupChannelSubs.subscribers[GroupChannelIdentifier{groupId: groupId, channelId: channelId}]
		if ok {
			for sub := range subscribers {
				go func(s chan []byte) {
					s <- payload
				}(sub)
			}
		}
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
		activeSockets:    NewPubSub[string](),
		groupChannelSubs: NewPubSub[GroupChannelIdentifier](),
		authHandler:      NewPublicKeysHandler(audience),
		db:               db,
	}
}

func NewPubSub[T comparable]() PubSub[T] {
	return PubSub[T]{
		subscribers: map[T]map[chan []byte]bool{},
	}
}

// WARN: Is it reliable to compare channels for equality directly ?

// func (self *GroupWebSocketsState) NewSocket(username string, writer chan []byte) {
// 	self.mu.Lock()
// 	defer self.mu.Unlock()

// 	writers, found := self.sockets[username]
// 	if found {

// 		for _, existingWriter := range writers {
// 			if existingWriter == writer {
// 				return
// 			}
// 		}

// 		writers = append(writers, writer)
// 	} else {
// 		writers = []chan []byte{writer}
// 		self.sockets[username] = writers
// 	}
// }

// func (self *GroupWebSocketsState) SocketClosed(username string, writer chan []byte) {
// 	self.mu.Lock()
// 	defer self.mu.Unlock()

// 	writers, found := self.sockets[username]
// 	if found {
// 		for idx, existingWriter := range writers {
// 			if existingWriter == writer {
// 				writers[idx] = writers[len(writers)-1]
// 				writers = writers[:len(writers)-1]

// 				if len(writers) == 0 {
// 					delete(self.sockets, username)
// 				}
// 				break
// 			}
// 		}
// 	}
// }

// func (self *GroupWebSocketsState) SendMessageToUsers(users []string, payload []byte) {
// 	self.mu.RLock()
// 	defer self.mu.RUnlock()
// 	var wg sync.WaitGroup
// 	for _, user := range users {
// 		self.SendMessageToUser(user, payload, &wg)
// 	}

// 	go wg.Wait()
// }

// func (self *GroupWebSocketsState) SendMessageToUser(username string, payload []byte, wg *sync.WaitGroup) {
// 	sockets, ok := self.sockets[username]
// 	if ok {
// 		for _, socket := range sockets {
// 			wg.Add(1)
// 			go func() {
// 				defer wg.Done()
// 				socket <- payload
// 			}()
// 		}
// 	}

// }

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

	var subscribedTopics = make([]GroupChannelIdentifier, 0)

	removeFromActiveSockets := func() {
		state.activeSockets.mu.Lock()
		defer state.activeSockets.mu.Unlock()
		sockets, ok := state.activeSockets.subscribers[verifiedToken.Username]
		if ok {
			delete(sockets, messageSink)
			if len(sockets) == 0 {
				delete(state.activeSockets.subscribers, verifiedToken.Username)
			}
		}

	}
	unsubscribeAllChannels := func() {
		state.groupChannelSubs.mu.Lock()
		defer state.groupChannelSubs.mu.Unlock()
		for _, topic := range subscribedTopics {
			subscribers, ok := state.groupChannelSubs.subscribers[topic]
			if ok {
				delete(subscribers, messageSink)
				if len(subscribers) == 0 {
					delete(state.groupChannelSubs.subscribers, topic)
				}
			} else {
				log.Printf("No topic exists group: %d channel: %d", topic.groupId, topic.channelId)
			}
		}

		subscribedTopics = subscribedTopics[:0]
	}

	subscribeToChannels := func(channels []*protos.GroupChannel) {
		state.groupChannelSubs.mu.Lock()
		defer state.groupChannelSubs.mu.Unlock()
		for _, channel := range channels {
			key := GroupChannelIdentifier{
				groupId:   int(channel.GetGroupId()),
				channelId: int(channel.GetChannelId()),
			}
			var subscribers, ok = state.groupChannelSubs.subscribers[key]
			if !ok {
				subscribers = make(map[chan []byte]bool)
			}

			subscribers[messageSink] = true
		}

	}

	defer func() {
		conn.Close()
		if verifiedToken != nil {
			unsubscribeAllChannels()
			removeFromActiveSockets()
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

							state.activeSockets.mu.Lock()
							defer state.activeSockets.mu.Unlock()
							var subscribed, ok = state.activeSockets.subscribers[verifiedToken.Username]
							if !ok {
								subscribed = make(map[chan []byte]bool)
							}
							subscribed[messageSink] = true

							chats, err := GetAllGroupChatsUserIn(state.db, verifiedToken.Username)
							if err != nil {
								log.Print(err)
								break
							}

							payload, err := proto.Marshal(&protos.ServerMessage{
								Message: &protos.ServerMessage_GroupChats{GroupChats: chats},
							})

							if err != nil {
								log.Print(err)
								break
							}

							messageSink <- payload
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
					state.AddGroupMessage(groupMsg, verifiedToken)
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
			case *protos.ClientMessage_CurrentGroup:
				{
					groupId := x.CurrentGroup
					channels, err := GetChannelsUserHasAccessTo(state.db, int(groupId), verifiedToken.Username)

					if err != nil {
						log.Print(err)
						continue
					}

					unsubscribeAllChannels()
					subscribeToChannels(channels)

					body, err := proto.Marshal(&protos.ServerMessage{
						Message: &protos.ServerMessage_GroupChat{
							GroupChat: &protos.GroupChat{
								Channels: channels,
								GroupId:  groupId,
							},
						},
					})

					if err != nil {
						log.Print(err)
						break
					}

					messageSink <- body
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
	http.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {

	})
	http.Handle("/", http.StripPrefix("/", fs))
	http.HandleFunc("/egg", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Egg"))
	})

	log.Println("Serving at ", addr)
	http.ListenAndServe(addr, nil)
}
