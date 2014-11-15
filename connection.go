package main

import (
	"strconv"

	"github.com/gorilla/websocket"
)

type UserConnection struct {
	UserId int
	ws     *websocket.Conn
}

func (user_connection *UserConnection) Listen() {
	defer func() {
		user_connection.ws.Close()
		if user_connection.UserId != 0 {
			registry.Unregister(user_connection)
		}
	}()

	for {
		_, message, err := user_connection.ws.ReadMessage()
		if err != nil {
			break
		}
		user_connection.UserId, err = strconv.Atoi(string(message))
		if err != nil {
			continue
		}
		registry.Register(user_connection)
	}
}
