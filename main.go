package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	. "firefly/pkg/utils"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: true,
}

func GetUsernameFromAuthorizationToken(token string) *string {
	log.Println("WARN: You are using unsafe authorization tokens, implement your own secure logic")
	username := strings.TrimSpace(token)
	if len(username) == 0 {
		return nil
	}
	return &username
}

func handleWebSocket(conn *websocket.Conn, server *GroupServer) {

	var username *string = nil
	userMessageSink := make(chan []byte)

	defer func() {
		if username != nil {
			server.MemberOffline(*username, userMessageSink)
			log.Println("User disconnected", *username)
		}
		conn.Close()
		close(userMessageSink)
	}()

	go func() {
		for msg := range userMessageSink {
			if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				log.Println("Failed to write message to socket")
				break
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		socketMessage := Message{}
		if err = json.Unmarshal(msg, &socketMessage); err != nil {
			log.Println("WARN: Received Invalid Socket Message ", string(msg))
			continue
		}

		if username == nil {
			if socketMessage.MsgType == AuthorizationMsgType {
				username = GetUsernameFromAuthorizationToken(socketMessage.Payload)
				if username != nil {
					if !server.MemberOnline(*username, userMessageSink) {
						log.Println("User doesn't exist in server ", *username)
						username = nil
					}
				}
			}
		} else {
			server.HandleMessage(socketMessage)
		}
	}
}

func websocketHandler(w http.ResponseWriter, r *http.Request, server *GroupServer) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	go handleWebSocket(conn, server)

}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
	_ = r
}

func main() {

	// runtime.GOMAXPROCS(1)

	server := NewGroupServer()

	server.AddMember("ash")
	// server.AddMember("dana")
	// server.AddChannel("chan0")
	// server.AddMemberToChannel("ash", "chan0")
	// server.AddMemberToChannel("dana", "chan0")

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { websocketHandler(w, r, &server) })

	http.ListenAndServe("0.0.0.0:8982", nil)
}
