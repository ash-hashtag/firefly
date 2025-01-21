package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"strings"

	. "firefly/pkg/utils"
	"github.com/gorilla/websocket"
)

func main() {

	colorReset := "\033[0m"
	// colorRed := "\033[31m"
	// colorGreen := "\033[32m"
	// colorYellow := "\033[33m"
	// colorBlue := "\033[34m"
	// colorPurple := "\033[35m"
	colorCyan := "\033[36m"
	// colorWhite := "\033[37m"
	username := os.Getenv("USERNAME")
	if len(username) == 0 {
		log.Fatalf("Username is empty")
	}
	reader := bufio.NewReader(os.Stdin)

	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:8982/ws", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	go func() {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				break
			}
			println(colorCyan + string(msg) + colorReset)
		}

	}()

	{
		msg := Message{
			MsgType: AuthorizationMsgType,
			Payload: username,
			By:      username,
		}

		if err := c.WriteJSON(msg); err != nil {
			log.Fatalf("%v", err)
		}
	}

	for {
		l, isPrefix, err := reader.ReadLine()
		if err != nil {
			log.Fatalf("%v", err)
		}
		if isPrefix {
			log.Fatalf("We don't handle long lines yet")
		}

		line := strings.TrimSpace(string(l))

		if line == "help" {
			println("===================help=======================")
			println("send     <channel_name> <msg>")
			println("add chan <channel_name>")
			println("add mem  <username> [channel_name]")
			println("del mem  <username> [channel_name]")
			println("channel  <channel_name>")
			println("channels")
			println("members")
			println("quit|exit")
			println("=============================================")
			continue
		}

		if line == "quit" || line == "exit" {
			break
		}

		if line == "channels" || line == "members" {
			msg := Message{
				MsgType: InfoMsgType,
				Payload: line,
				By:      username,
			}

			if err := c.WriteJSON(msg); err != nil {
				log.Fatalf("%v", err)
			}

			continue
		}

		if strings.HasPrefix(line, "channel ") {
			msg := Message{
				MsgType: InfoMsgType,
				Payload: line,
				By:      username,
			}

			if err := c.WriteJSON(msg); err != nil {
				log.Fatalf("%v", err)
			}

			continue
		}

		if strings.HasPrefix(line, "send ") {
			line = strings.TrimSpace(line[5:])

			channel_name, msg, found := strings.Cut(line, " ")
			if !found {
				continue
			}

			payload, _ := json.Marshal(GroupChatMessage{
				ChannelName: channel_name,
				Content:     msg,
			})

			message := Message{
				MsgType: ChatMessageMsgType,
				Payload: string(payload),
				By:      username,
			}

			if err := c.WriteJSON(message); err != nil {
				log.Fatalf("%v", err)
			}

			continue

		}
		if strings.HasPrefix(line, "add ") {
			line = strings.TrimSpace(line[4:])
			if strings.HasPrefix(line, "chan ") {
				channel_name := strings.TrimSpace(line[5:])

				message := Message{
					MsgType: AddChannelMsgType,
					Payload: channel_name,
					By:      username,
				}

				if err := c.WriteJSON(message); err != nil {
					log.Fatalf("%v", err)
				}

			} else if strings.HasPrefix(line, "mem ") {
				line = strings.TrimSpace(line[4:])
				member_name, channel_name, found := strings.Cut(line, " ")

				if found {
					g_msg := GroupChatMessage{
						ChannelName: channel_name,
						Content:     member_name,
					}

					payload, _ := json.Marshal(g_msg)

					message := Message{
						MsgType: AddMemberToChannelMsgType,
						Payload: string(payload),
						By:      username,
					}

					if err := c.WriteJSON(message); err != nil {
						log.Fatalf("%v", err)
					}

				} else {

					message := Message{
						MsgType: AddMemberMsgType,
						Payload: member_name,
						By:      username,
					}

					if err := c.WriteJSON(message); err != nil {
						log.Fatalf("%v", err)
					}

				}
			}

			continue
		}
		if strings.HasPrefix(line, "del ") {
			line = strings.TrimSpace(line[4:])
			// splits := strings.SplitN(line, " ", 2)
			// member_name := splits[0]

			member_name, channel_name, found := strings.Cut(line, " ")
			if found {
				// channel_name := splits[1]
				g_msg := GroupChatMessage{
					ChannelName: channel_name,
					Content:     member_name,
				}

				payload, _ := json.Marshal(g_msg)

				message := Message{
					MsgType: KickMemberOnlyFromChannelMsgType,
					Payload: string(payload),
					By:      username,
				}

				if err := c.WriteJSON(message); err != nil {
					log.Fatalf("%v", err)
				}

			} else {
				message := Message{
					MsgType: KickMemberMsgType,
					Payload: member_name,
					By:      username,
				}

				if err := c.WriteJSON(message); err != nil {
					log.Fatalf("%v", err)
				}

			}

			continue
		}
	}
}
