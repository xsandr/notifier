package main

import (
	"flag"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"text/template"

	"github.com/gorilla/websocket"
)

var (
	debug    = flag.Bool("debug", false, "Debug allow serve index page")
	ssl      = flag.Bool("ssl", false, "SSL usage")
	addr     = flag.String("addr", ":5000", "ws service address")
	rabbit   = flag.String("rabbit", "amqp://guest:guest@localhost:5672/", "AMQP URI")
	exchange = flag.String("exchange", "notifications", "Durable, non-auto-deleted AMQP exchange name")
	queue    = flag.String("queue", "notifications", "Queue name")
	routing  = flag.String("routing key", "user.*", "Routing key for queue")
	certFile = flag.String("cert", "", "Cert for TLS")
	keyFile  = flag.String("keyfile", "", "Key for TLS")
	re       = regexp.MustCompile("user.(\\d+)")

	homeTempl = template.Must(template.ParseFiles("index.html"))
	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
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
			log.Printf("User %d leave ", user_connection.UserId)
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
		log.Printf("User %d registred ", user_connection.UserId)
	}
}

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
	if _, ok := registry.connections[uc.UserId]; ok == false {
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
	registry.Unlock()
}

var registry = &Regestry{connections: make(map[int][]*UserConnection)}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	user_connection := &UserConnection{ws: ws}
	go user_connection.Listen()
}

func serveMain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	homeTempl.Execute(w, r.Host)
}

func main() {
	flag.Parse()
	go registry.ListenRabbit()
	if *debug {
		http.HandleFunc("/", serveMain)
	}
	http.HandleFunc("/ws", serveWs)
	if *ssl {
		http.ListenAndServeTLS(*addr, *certFile, *keyFile, nil)
	} else {
		http.ListenAndServe(*addr, nil)
	}
}
