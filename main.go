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
	addr     = flag.String("addr", ":7788", "ws service address")
	rabbit   = flag.String("rabbit", "amqp://guest:guest@localhost:5672/", "AMQP URI")
	exchange = flag.String("exchange", "notifications", "Durable, non-auto-deleted AMQP exchange name")
	queue    = flag.String("queue", "notifications", "Queue name")
	routing  = flag.String("routing key", "user.*", "Routing key for queue")
	re       = regexp.MustCompile("user.(\\d+)")

	homeTempl = template.Must(template.ParseFiles("index.html"))
	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

type UserConnection struct {
	UserId int
	ws     *websocket.Conn
}

func (user_connection *UserConnection) Listen() {
	defer user_connection.ws.Close()
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
	registry.Unregister(user_connection.UserId)
	log.Printf("User %d leave ", user_connection.UserId)
}

type Regestry struct {
	sync.RWMutex
	connections map[int]*UserConnection
}

func (registry Regestry) ListenRabbit() {
	for message := range GetMessagesChannel() {
		message.Ack(false)
		matches := re.FindStringSubmatch(message.RoutingKey)
		if len(matches) == 0 {
			continue
		}
		user_id, _ := strconv.Atoi(matches[len(matches)-1])
		log.Printf("Message for user  %d: '%s'", user_id, string(message.Body))
		ws_connection, ok := registry.connections[user_id]
		if ok == false {
			log.Printf("But user %d not connected", user_id)
			continue
		}
		ws_connection.ws.WriteMessage(websocket.TextMessage, message.Body)
	}
}

func (registry Regestry) Register(uc *UserConnection) {
	registry.Lock()
	registry.connections[uc.UserId] = uc
	registry.Unlock()
}

func (registry Regestry) Unregister(user_id int) {
	registry.Lock()
	delete(registry.connections, user_id)
	registry.Unlock()
}

var registry = Regestry{connections: make(map[int]*UserConnection)}

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
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
