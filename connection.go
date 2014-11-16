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
		registry.Unregister(user_connection)
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
		messages_channel, connection := GetUndeliveredMessage(user_connection.UserId)
		defer connection.Close()

		if is_new_user := registry.Register(user_connection); is_new_user {
			messages := make(chan []byte)
			go func() {
				defer close(messages)
				for m := range messages_channel {
					messages <- m.Body
					m.Ack(false)
				}
			}()
			go registry.SendUndeliveredMessages(user_connection.UserId, messages)
		}
	}
}
