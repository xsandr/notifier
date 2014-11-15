package main

import (
	"log"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

type Regestry struct {
	sync.RWMutex
	connections map[int][]*UserConnection
}

func (registry *Regestry) ListenRabbit() {
	for message := range GetMessagesChannel() {
		message.Ack(false)
		matches := re.FindStringSubmatch(message.RoutingKey)
		if len(matches) == 0 {
			continue
		}
		user_id, _ := strconv.Atoi(matches[len(matches)-1])
		ws_connection, ok := registry.GetConnection(user_id)
		log.Printf("Message for user(online - %v) %d: '%s'", ok, user_id, string(message.Body))
		if ok == false {
			continue
		}
		ws_connection.ws.WriteMessage(websocket.TextMessage, message.Body)
	}
}

func (registry *Regestry) GetConnection(user_id int) (*UserConnection, bool) {
	registry.Lock()
	defer registry.Unlock()
	ws_connections, ok := registry.connections[user_id]
	if ok == false || len(ws_connections) == 0 {
		return nil, false
	}
	return ws_connections[0], true
}

func (registry *Regestry) Register(uc *UserConnection) {
	registry.Lock()
	defer registry.Unlock()
	connectionsCount.Add(1)
	if _, ok := registry.connections[uc.UserId]; ok == false {
		usersCount.Add(1)
		registry.connections[uc.UserId] = make([]*UserConnection, 0)
	}
	registry.connections[uc.UserId] = append(registry.connections[uc.UserId], uc)
}

func (registry *Regestry) Unregister(uc *UserConnection) {
	registry.Lock()
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
	}
	registry.Unlock()
}
