package main

import (
	"log"
	"sync"
)

// Registry contains all web-socket connections
type Registry struct {
	sync.RWMutex
	connections map[int][]*UserConnection
	Consumer    *Consumer
}

// NewRegistry creates new Registry
func NewRegistry() *Registry {
	consumer := NewConsumer()
	go consumer.Run()
	return &Registry{
		connections: make(map[int][]*UserConnection),
		Consumer:    consumer,
	}
}

// ListenAndSendMessages gets all messages from RabbitMQ and sends them to recipients
func (r *Registry) ListenAndSendMessages() {
	for message := range r.GetMessages() {
		if userConnection, ok := r.GetConnection(message.UID); ok {
			userConnection.Send(message)
		} else if isTTLEnabled {
			r.SendUndeliveredMessages(message)
		}
	}
}

// GetMessages returns Messages channel
func (r *Registry) GetMessages() chan Message {
	return r.Consumer.Messages
}

// SendUndeliveredMessages publish message for offline users
func (r *Registry) SendUndeliveredMessages(m Message) {
	r.Consumer.PublishUndeliveredMessage(m)
}

// GetConnection returns UserConnection for uid
func (r *Registry) GetConnection(uid int) (*UserConnection, bool) {
	r.Lock()
	defer r.Unlock()

	if wsConnetions, ok := r.connections[uid]; ok {
		return wsConnetions[0], true
	}
	return nil, false
}

// Register add user connection to Registry
func (r *Registry) Register(uc *UserConnection) {
	r.Lock()
	defer r.Unlock()
	_, ok := r.connections[uc.UID]
	if ok == false {
		r.connections[uc.UID] = make([]*UserConnection, 0)
	}
	r.connections[uc.UID] = append(r.connections[uc.UID], uc)
	log.Printf("User %d registered", uc.UID)
	log.Printf("Connections %d", len(r.connections))

	if ok == false && isTTLEnabled {
		log.Printf("Go to check undelivered message for user %d", uc.UID)
		r.Consumer.GetUndeliveredMessage(uc.UID)
	}
}

// Unregister remove user ws connection from Registry, when user leaves
func (r *Registry) Unregister(uc *UserConnection) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.connections[uc.UID]; ok {
		var index int
		userConnections := r.connections[uc.UID]
		for i := 0; i < len(userConnections); i++ {
			if userConnections[i] == uc {
				index = i
				break
			}
		}
		r.connections[uc.UID] = append(userConnections[:index], userConnections[index+1:]...)

		if len(r.connections[uc.UID]) == 0 {
			delete(r.connections, uc.UID)
			log.Printf("User %d leave", uc.UID)
		}
	}
}

func (r *Registry) GetOnlineUsers() []int {
	r.Lock()
	defer r.Unlock()
	users := make([]int, 0)
	for uid, _ := range r.connections {
		users = append(users, uid)
	}
	return users
}
