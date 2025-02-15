package utils

import "sync"

type GroupWebSocketsState struct {
	sockets map[string][]chan []byte // username -> multiple channels
	mu      sync.RWMutex
}

func NewGroupWebSocketsState() GroupWebSocketsState {
	return GroupWebSocketsState{
		sockets: make(map[string][]chan []byte),
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
