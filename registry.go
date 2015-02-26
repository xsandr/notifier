package main

import (
	"log"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
)

type Regestry struct {
	sync.RWMutex
	connection  *amqp.Connection
	connections map[int][]*UserConnection
}

func (registry *Regestry) ListenRabbit() {
	message_delivery, connection := GetMessagesChannel()
	registry.connection = connection
	for message := range message_delivery {
		message.Ack(false)
		matches := re.FindStringSubmatch(message.RoutingKey)
		if len(matches) == 0 {
			continue
		}
		user_id, _ := strconv.Atoi(matches[len(matches)-1])
		ws_connection, ok := registry.GetConnection(user_id)
		log.Printf("Message for user(is online - %v) %d: '%s'", ok, user_id, string(message.Body))
		if ok {
			ws_connection.ws.WriteMessage(websocket.TextMessage, message.Body)
		} else if *defatul_ttl > 0 {
			ttl := *defatul_ttl
			if ttl_header, ok := message.Headers["ttl"]; ok && ttl_header != nil {
				if ttl_int, ok := ttl_header.(int32); ok {
					ttl = int64(ttl_int)
				}
			}
			PublishUndeliveredMessage(connection, user_id, message.Body, ttl)
		} else {
			log.Print("but TTL disabled")
		}
	}
}

// Sending undelivered messages through WS
func (registry *Regestry) SendUndeliveredMessages(user_id int, messages chan []byte) {
	for message := range messages {
		if ws_connection, ok := registry.GetConnection(user_id); ok {
			log.Printf("Undelivered message for user(is online - %v) %d: '%s'", ok, user_id, string(message))
			ws_connection.ws.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func (registry *Regestry) GetConnection(user_id int) (*UserConnection, bool) {
	registry.Lock()
	defer registry.Unlock()
	ws_connections, ok := registry.connections[user_id]
	log.Printf("User %d have %d active connection", user_id, len(ws_connections))
	if ok == false || len(ws_connections) == 0 {
		return nil, false
	}
	return ws_connections[0], true
}

func (registry *Regestry) Register(uc *UserConnection) bool {
	registry.Lock()
	defer registry.Unlock()
	connectionsCount.Add(1)
	log.Printf("User %d: register", uc.UserId)
	first_connection := false
	if _, ok := registry.connections[uc.UserId]; ok == false {
		usersCount.Add(1)
		first_connection = true
		registry.connections[uc.UserId] = make([]*UserConnection, 0)
	}
	registry.connections[uc.UserId] = append(registry.connections[uc.UserId], uc)
	return first_connection
}

func (registry *Regestry) Unregister(uc *UserConnection) {
	registry.Lock()
	log.Printf("User %d: unregister", uc.UserId)
	if _, ok := registry.connections[uc.UserId]; ok {
		var index int = 0
		for i := 0; i < len(registry.connections[uc.UserId]); i++ {
			if registry.connections[uc.UserId][i] == uc {
				index = i
				break
			}
		}
		registry.connections[uc.UserId] = append(registry.connections[uc.UserId][:index], registry.connections[uc.UserId][index+1:]...)

		connectionsCount.Add(-1)
		if len(registry.connections[uc.UserId]) == 0 {
			usersCount.Add(-1)
			delete(registry.connections, uc.UserId)
		}
	}
	registry.Unlock()
}
