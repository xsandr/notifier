package main

import (
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type UserConnection struct {
	UserId int
	ws     *websocket.Conn
	active bool
}

func (user_connection *UserConnection) Listen() {
	defer func() {
		user_connection.ws.Close()
		if user_connection.active {
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
		user_connection.active = true

		messages_channel, connection := GetUndeliveredMessage(user_connection.UserId)
		if is_new_user := registry.Register(user_connection); is_new_user {
			messages := make(chan []byte)
			go func() {
				for {
					select {
					case m := <-messages_channel:
						messages <- m.Body
						m.Ack(false)
					case <-time.After(20 * time.Second):
						connection.Close()
						close(messages)
						return
					}
				}
			}()
			go registry.SendUndeliveredMessages(user_connection.UserId, messages)
		}
	}
}
