package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

type GroupChannel struct {
	ChannelName    string
	ChannelMembers map[string]time.Time
}

func NewGroupChannel(name string) GroupChannel {
	return GroupChannel{
		ChannelName:    name,
		ChannelMembers: map[string]time.Time{},
	}
}

func (channel *GroupChannel) AddMember(name string) {
	channel.ChannelMembers[name] = time.Now()
}

type GroupServer struct {
	channels      map[string]*GroupChannel
	members       map[string]time.Time
	onlineMembers map[string]chan []byte
}

func NewGroupServer() GroupServer {
	return GroupServer{
		channels:      map[string]*GroupChannel{},
		members:       map[string]time.Time{},
		onlineMembers: map[string]chan []byte{},
	}
}

func (server *GroupServer) HandleMessage(msg Message) {
	switch msg.MsgType {
	case ChatMessageMsgType:
		gMsg := GroupChatMessage{}
		if err := json.Unmarshal([]byte(msg.Payload), &gMsg); err != nil {
			log.Println("WARN: Received Invalid Chat Message")
			return
		}

		rawMsg := []byte(fmt.Sprintf("[%s] %s: %s", gMsg.ChannelName, msg.By, gMsg.Content))
		server.SendToChannelMembers(rawMsg, gMsg.ChannelName, &msg.By)

	case AddChannelMsgType:
		channelName := msg.Payload
		server.AddChannel(channelName)
		rawMsg := []byte(fmt.Sprintf("server@all : Channel Created %s By %s", channelName, msg.By))
		server.SendAll(rawMsg)

	case AddMemberMsgType:
		memberName := msg.Payload
		server.AddMember(memberName)
		rawMsg := []byte(fmt.Sprintf("server@all : %s Joined the server by %s", memberName, msg.By))
		server.SendAll(rawMsg)

	case AddMemberToChannelMsgType:
		gMsg := GroupChatMessage{}
		if err := json.Unmarshal([]byte(msg.Payload), &gMsg); err != nil {
			log.Println("WARN: Invalid Payload")
			return
		}
		memberName := gMsg.Content
		channelName := gMsg.ChannelName
		server.AddMemberToChannel(memberName, gMsg.ChannelName)
		rawMsg := []byte(fmt.Sprintf("server@all : %s Joined the Channel %s by %s", memberName, channelName, msg.By))
		server.SendToChannelMembers(rawMsg, channelName, nil)

	case InfoMsgType:
		if msg.Payload == "channels" {
			s := "Channels:\n"
			for channelName := range server.channels {
				s += channelName + "\n"
			}
			rawMsg := []byte(s[:len(s)-1])
			server.SendTo(rawMsg, msg.By)

		} else if msg.Payload == "members" {
			s := "Members:"
			for member, lastSeen := range server.members {
				s += "\n"
				s += member

				if _, ok := server.onlineMembers[member]; ok {
					s += " status: online"
				} else {
					s += " status: offline, last seen " + lastSeen.Format(time.RFC850)
				}

			}
			rawMsg := []byte(s)
			server.SendTo(rawMsg, msg.By)
		} else {

			if channelName, ok := strings.CutPrefix(msg.Payload, "channel "); ok {
				if channel, ok := server.channels[channelName]; ok {
					s := "Channel: " + channelName
					for member := range channel.ChannelMembers {
						s += "\n"
						s += member
						if _, ok := server.onlineMembers[member]; ok {
							s += " status: online"
						} else {
							lastSeen := server.members[member]
							s += " status: offline, last seen " + lastSeen.Format(time.RFC850)
						}

					}

					server.SendTo([]byte(s), msg.By)
				}
			}

		}
	default:
		log.Println("WARN: Received Invalid Socket Message Type ", msg.MsgType)

	}
}

func (server *GroupServer) SendTo(rawMsg []byte, by string) {
	if userChan, ok := server.onlineMembers[by]; ok {
		go func() {
			userChan <- rawMsg
		}()
	}
}

func (server *GroupServer) SendAll(rawMsg []byte) {
	for _, userChan := range server.onlineMembers {
		go func() {
			userChan <- rawMsg
		}()
	}

}

func (server *GroupServer) SendToChannelMembers(rawMsg []byte, channelName string, except *string) {
	channel, ok := server.channels[channelName]
	if !ok {
		return
	}

	if len(server.onlineMembers) < len(channel.ChannelMembers) {
		for channelMember := range channel.ChannelMembers {
			if except != nil && *except == channelMember {
				continue
			}

			server.SendTo(rawMsg, channelMember)

			// if userChan, ok := server.onlineMembers[channelMember]; ok {
			// 	go func() {
			// 		userChan <- rawMsg
			// 	}()
			// }
		}
	} else {
		for onlineUser, userChan := range server.onlineMembers {
			if except != nil && *except == onlineUser {
				continue
			}

			if _, ok := channel.ChannelMembers[onlineUser]; ok {

				go func() {
					userChan <- rawMsg
				}()
			}

		}
	}

}

func (server *GroupServer) MemberOnline(name string, userChan chan []byte) bool {
	if _, ok := server.members[name]; !ok {
		return false
	}
	server.onlineMembers[name] = userChan
	server.members[name] = time.Now()
	return true
}
func (server *GroupServer) MemberOffline(name string) bool {
	if _, ok := server.members[name]; !ok {
		return false
	}
	delete(server.onlineMembers, name)
	server.members[name] = time.Now()
	return true
}

func (server *GroupServer) AddMember(name string) {
	server.members[name] = time.Now()
	log.Println("Added Member ", name)
}
func (server *GroupServer) AddMemberToChannel(username, channelName string) bool {

	_, ok := server.members[username]
	if !ok {
		log.Printf("WARN: No member in server with name %s", username)
		return false
	}

	channel, ok := server.channels[channelName]
	if !ok {
		log.Printf("WARN: No channel in server with name %s", channelName)
		return false
	}
	channel.AddMember(username)
	log.Println("Added Member ", username, " To Channel", channel.ChannelName)

	return true
}
func (server *GroupServer) AddChannel(name string) {
	channel := NewGroupChannel(name)
	server.channels[name] = &channel
}
