package main

import (
	"strconv"

	"github.com/gorilla/websocket"
)

// UserConnection represents user ws-connection and his UID
type UserConnection struct {
	UID int
	ws  *websocket.Conn
}

// NewUserConnection constructor for UserConnection
func NewUserConnection(ws *websocket.Conn) *UserConnection {
	return &UserConnection{ws: ws}
}

// Send method deliver message on websocket
func (u *UserConnection) Send(m Message) {
	u.ws.WriteMessage(websocket.TextMessage, []byte(m.Message))
}

// Listen method listens ws-connection and tries to get user UID
func (u *UserConnection) Listen() {
	defer func() {
		u.ws.Close()
		registry.Unregister(u)
	}()

	for {
		_, message, err := u.ws.ReadMessage()
		if err != nil {
			break
		}
		uid, err := strconv.Atoi(string(message))
		if err != nil {
			continue
		}
		u.UID = uid
		registry.Register(u)
	}
}
