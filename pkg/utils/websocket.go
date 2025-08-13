package utils

import (
	"context"
	"firefly/pkg/protos"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

type GroupChannelIdentifier struct {
	groupId, channelId int
}

type WebSocketsState struct {
	authHandler      PublicKeysHandler
	db               *pgxpool.Pool
	groupChannelSubs PubSub[GroupChannelIdentifier] // subscribers
	activeUsers      PubSub[string]
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
		subscriber <- payload
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

func (self *WebSocketsState) RemoveUser(removeUser *protos.RemoveUser, verifiedToken *VerifiedToken) {
	if verifiedToken == nil || 0 == len(verifiedToken.Username) {
		log.Printf("ERROR: User is not authenticated")
		return
	}

	username := removeUser.GetUsername()
	channel := int(removeUser.GetGroupId())
	groupId := int(removeUser.GetGroupId())

	RemoveUserFromChannel(self.db, username, groupId, channel, verifiedToken.Username)
}

func (self *WebSocketsState) AddUser(groupId int, addUser *protos.AddUser, verifiedToken *VerifiedToken) {

	if verifiedToken == nil || 0 == len(verifiedToken.Username) {
		log.Printf("ERROR: User is not authenticated")
		return
	}

	username := addUser.GetUsername()
	channel := int(addUser.GetChannelId())
	role := int(addUser.GetRole())

	AddUserToChannel(self.db, username, groupId, channel, role, verifiedToken.Username)

}

func (self *WebSocketsState) HandleRequest(msg *protos.Request, verifiedToken *VerifiedToken) *protos.Response {

	requestId := msg.GetId()
	response := &protos.Response{
		Id: requestId,
	}
	switch x := msg.Message.(type) {

	case *protos.Request_GetGroupMembers:
		users, err := GetUsersInChannel(self.db, int(x.GetGroupMembers.GetGroupId()), int(x.GetGroupMembers.GetChannelId()))
		if err != nil {
			log.Print(err)
			return nil
		}

		self.activeUsers.mu.RLock()
		defer self.activeUsers.mu.RUnlock()
		for _, user := range users {
			_, isOnline := self.activeUsers.subscribers[user.GetUsername()]
			user.IsOnline = isOnline
		}

		response.Message = &protos.Response_GroupMembers{GroupMembers: &protos.GroupMembers{Members: users}}

		return response

	case *protos.Request_GetGroupMessages:

		before := x.GetGroupMessages.GetBefore()
		if len(before) != 16 {

			response.Message = &protos.Response_Error{Error: &protos.Error{Error: "Internal fields", Status: 400}}
			return response

		}
		var count = x.GetGroupMessages.GetCount()
		if count > 100 {
			count = 100
		}

		messages, err := GetGroupMessages(self.db, int(x.GetGroupMessages.GetGroupId()), int(x.GetGroupMessages.GetChannelId()), uuid.UUID([16]byte(before)), uint64(count))
		if err != nil {
			log.Print(err)
			response.Message = &protos.Response_Error{Error: &protos.Error{Error: "Internal error", Status: 500}}
			return response
		}
		response.Message =
			&protos.Response_GroupMessages{GroupMessages: &protos.GroupChannelMessages{Messages: messages}}
		return response

	case *protos.Request_AddUser:
		err := AddUserToChannel(self.db, x.AddUser.GetUsername(), int(x.AddUser.GetGroupId()), int(x.AddUser.GetChannelId()), int(x.AddUser.GetRole()), verifiedToken.Username)
		if err != nil {
			response.Message = &protos.Response_Error{Error: &protos.Error{Error: "unauthorized or group or channel doesn't exist", Status: 400}}
			return response

		}
		return response
	case *protos.Request_RemoveUser:
		err := RemoveUserFromChannel(self.db, x.RemoveUser.GetUsername(), int(x.RemoveUser.GetGroupId()), int(x.RemoveUser.GetChannelId()), verifiedToken.Username)
		if err != nil {
			response.Message = &protos.Response_Error{Error: &protos.Error{Error: "unauthorized or group or channel doesn't exist", Status: 400}}
			return response

		}
		return response
	case *protos.Request_AddChannel:
	case *protos.Request_DeleteChannel:
	case *protos.Request_GetUserMessages:
		before := x.GetUserMessages.GetBefore()
		if len(before) != 16 {
			response.Message = &protos.Response_Error{Error: &protos.Error{Error: "Invalid fields", Status: 400}}
			return response
		}
		var count = x.GetUserMessages.GetCount()
		if count > 100 {
			count = 100
		}

		other := x.GetUserMessages.GetFrom()

		messages, err := GetUserMessages(self.db, verifiedToken.Username, other, uuid.UUID([16]byte(before)), uint64(count))
		if err != nil {
			log.Print(err)
			response.Message = &protos.Response_Error{Error: &protos.Error{Error: "Internal error", Status: 500}}
			return response
		}

		response.Message = &protos.Response_UserMessages{UserMessages: messages}
		return response

	default:
		err := fmt.Sprintf("unexpected protos.isRequest_Message: %#v", x)
		response.Message = &protos.Response_Error{Error: &protos.Error{Status: 404, Error: err}}
		return response

	}

	return &protos.Response{Id: requestId}
}

func (self *WebSocketsState) AddGroupMessage(groupMsg *protos.GroupChannelMessage, verifiedToken *VerifiedToken) error {

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
				sub <- payload
			}
		}
	}

	return nil
}

func (self *WebSocketsState) SendUserMessage(msg *protos.UserMessage) error {
	id, err := InsertUserMessage(self.db, msg.GetFrom(), msg.GetTo(), msg.GetText())
	if err != nil {
		return err
	}

	msg.Id = id[:]
	payload, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	self.activeUsers.SendTo(msg.GetTo(), payload)
	self.activeUsers.SendTo(msg.GetFrom(), payload)
	return err
}

func NewGroupWebSocketsState() WebSocketsState {

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

	return WebSocketsState{
		activeUsers:      NewPubSub[string](),
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: true,
}

func WebsocketHandler(w http.ResponseWriter, r *http.Request, server *WebSocketsState) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	go handleWebSocket(conn, server)
}

func handleWebSocket(conn *websocket.Conn, state *WebSocketsState) {

	messageSink := make(chan []byte, 20)
	var verifiedToken *VerifiedToken

	var subscribedTopics = make([]GroupChannelIdentifier, 0)

	removeFromActiveSockets := func() {
		state.activeUsers.mu.Lock()
		defer state.activeUsers.mu.Unlock()
		sockets, ok := state.activeUsers.subscribers[verifiedToken.Username]
		if ok {
			delete(sockets, messageSink)
			if len(sockets) == 0 {
				delete(state.activeUsers.subscribers, verifiedToken.Username)
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
			subscribedTopics = append(subscribedTopics, key)
		}

	}

	defer func() {
		conn.Close()
		if verifiedToken != nil {
			unsubscribeAllChannels()
			removeFromActiveSockets()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	ctx, cancel := context.WithCancel(context.Background())

	writeFunc := func() {
		defer wg.Done()
	forLoop:
		for {
			select {
			case <-ctx.Done():
				return
			case payload := <-messageSink:
				if err := conn.WriteMessage(websocket.BinaryMessage, payload); err != nil {
					log.Printf("ERROR: writing message, %s", err)
					break forLoop
				}
			default:
				break forLoop
			}
		}
		cancel()
	}

	readFunc := func() {
		readerSink := make(chan []byte)
		go func() {
			for {
				msgType, msg, err := conn.ReadMessage()
				_ = msgType
				if err != nil {
					log.Println(err)
					break
				}
				readerSink <- msg
			}
		}()
	forLoop:
		for {
			select {
			case <-ctx.Done():
				return

			case msg := <-readerSink:
				{
					message := protos.ClientMessage{}
					err := proto.Unmarshal(msg, &message)
					if err != nil {
						log.Printf("received invalid message %s", err)
						continue forLoop
					}

					if verifiedToken == nil {

						switch x := message.Payload.(type) {

						case *protos.ClientMessage_AuthToken:
							{
								token := x.AuthToken.GetToken()
								verifiedToken = state.authHandler.VerifyToken(token)
								if verifiedToken != nil {
									state.activeUsers.Subscribe(verifiedToken.Username, messageSink)
								}
							}
						default:
							continue forLoop
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
									continue forLoop
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
								continue forLoop
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
								break forLoop
							}

							messageSink <- body
						}
					case *protos.ClientMessage_UserMessage:
						{
							userMsg := x.UserMessage
							userMsg.From = verifiedToken.Username
							if len(userMsg.GetTo()) == 0 || len(userMsg.GetText()) == 0 {
								continue forLoop
							}
							state.SendUserMessage(userMsg)
						}
					default:
					}
				}
			}
		}
		cancel()
	}

	go readFunc()
	go writeFunc()
	wg.Wait()
}

func StartServer(addr string) {
	state := NewGroupWebSocketsState()
	// fs := http.FileServer(http.Dir("./public"))

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { WebsocketHandler(w, r, &state) })
	// http.Handle("/", http.StripPrefix("/", fs))
	http.HandleFunc("/egg", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Egg"))
	})

	http.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {

		token := state.authHandler.VerifyTokenFromHeaders(r.Header)

		if token == nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid authorization header"))
			return
		}

		chats, err := GetAllGroupChatsUserIn(state.db, token.Username)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			return
		}

		body, err := proto.Marshal(chats)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			return
		}

		w.Header().Set("content-type", "application/x-protobuf; proto=lupyd.chats.GroupChats")

		w.Write(body)

	})
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		token := state.authHandler.VerifyTokenFromHeaders(r.Header)

		if token == nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid authorization header"))
			return
		}

		msgs, err := GetAllUserChats(state.db, token.Username)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			return
		}

		body, err := proto.Marshal(msgs)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			return
		}

		w.Header().Set("content-type", "application/x-protobuf; proto=lupyd.chats.UserMessages")

		w.Write(body)
	})

	http.HandleFunc("/group", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Only supports POST"))
			return
		}
		token := state.authHandler.VerifyTokenFromHeaders(r.Header)
		if token == nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(""))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			log.Println(err)
			return
		}

		var groupChat protos.GroupChat

		err = proto.Unmarshal(body, &groupChat)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			log.Println(err)
			return
		}
		if len(groupChat.GetName()) < 3 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Group name is not valid"))
			return
		}

		grpId, err := CreateGroupChat(state.db, groupChat.GetName(), token.Username)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			log.Println(err)
			return
		}

		groupChat.GroupId = int32(grpId)

		body, err = proto.Marshal(&groupChat)
		if err != nil {

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			log.Println(err)
			return
		}

		w.Write(body)

	})

	http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Only supports POST"))
			return
		}
		token := state.authHandler.VerifyTokenFromHeaders(r.Header)
		if token == nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(""))
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(""))
			log.Println(err)
			return
		}

		if !utf8.Valid(body) || isValidUsername(string(body)) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid Username"))
			return
		}

		chatId, err := CreateUserChat(state.db, token.Username, string(body))

		if !utf8.Valid(body) || isValidUsername(string(body)) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
			return
		}

		w.Write(chatId[:])
	})
	log.Println("Serving at ", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Println(err)
	}
}

func isValidUsername(s string) bool {
	if len(s) < 3 || len(s) > 32 {
		return false
	}

	for _, c := range s {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '-' {
			return false
		}
	}

	return true
}
