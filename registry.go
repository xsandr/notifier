package main

import (
	"log"
	"sync"
)

type Regestry struct {
	sync.RWMutex
	connections map[int][]*UserConnection
	Consumer    *Consumer
}

func NewRegistry() *Regestry {
	consumer := NewConsumer()
	go consumer.Run()
	return &Regestry{
		connections: make(map[int][]*UserConnection),
		Consumer:    consumer,
	}
}

func (r *Regestry) ListenAndSendMessages() {
	for message := range r.GetMessages() {
		if user_connection, ok := r.GetConnection(message.UID); ok {
			user_connection.Send(message)
		} else if is_ttl_enabled {
			r.SendUndeliveredMessages(message)
		}
	}
}

func (r *Regestry) GetMessages() chan Message {
	return r.Consumer.Messages
}

func (r *Regestry) SendUndeliveredMessages(m Message) {
	r.Consumer.PublishUndeliveredMessage(m)
}

// Returns user connection, and
func (r *Regestry) GetConnection(user_id int) (*UserConnection, bool) {
	r.Lock()
	defer r.Unlock()

	if ws_connections, ok := r.connections[user_id]; ok {
		return ws_connections[0], true
	}
	return nil, false
}

func (r *Regestry) Register(uc *UserConnection) {
	r.Lock()
	defer r.Unlock()
	_, ok := r.connections[uc.UID]
	if ok == false {
		r.connections[uc.UID] = make([]*UserConnection, 0)
	}
	r.connections[uc.UID] = append(r.connections[uc.UID], uc)
	log.Printf("User %d registered", uc.UID)
	log.Printf("Connections %d", len(r.connections))

	if ok == false && is_ttl_enabled {
		log.Printf("Go to check undelivered message for user %d", uc.UID)
		r.Consumer.GetUndeliveredMessage(uc.UID)
	}
}

func (r *Regestry) Unregister(uc *UserConnection) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.connections[uc.UID]; ok {
		var index int = 0
		user_connections := r.connections[uc.UID]
		for i := 0; i < len(user_connections); i++ {
			if user_connections[i] == uc {
				index = i
				break
			}
		}
		r.connections[uc.UID] = append(user_connections[:index], user_connections[index+1:]...)

		if len(r.connections[uc.UID]) == 0 {
			delete(r.connections, uc.UID)
			log.Printf("User %d leave", uc.UID)
		}
	}
}
