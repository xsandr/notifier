package main

import (
	"strconv"

	"github.com/gorilla/websocket"
)

type UserConnection struct {
	UID int
	ws  *websocket.Conn
}

func NewUserConnection(ws *websocket.Conn) *UserConnection {
	return &UserConnection{ws: ws}
}

func (u *UserConnection) Send(m Message) {
	u.ws.WriteMessage(websocket.TextMessage, []byte(m.Message))
}

func (uc *UserConnection) Listen() {
	defer func() {
		uc.ws.Close()
		registry.Unregister(uc)
	}()

	for {
		_, message, err := uc.ws.ReadMessage()
		if err != nil {
			break
		}
		uid, err := strconv.Atoi(string(message))
		if err != nil {
			continue
		}
		uc.UID = uid
		registry.Register(uc)
	}
}
