package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/gorilla/websocket"
)

var (
	addr      = flag.String("addr", ":7788", "ws service address")
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

// Connect to rabbitmq and listen new messages for user_id
func (user_connection *UserConnection) ListenNotification() {
}

func (user_connection *UserConnection) Listen() {
	defer user_connection.ws.Close()
	for {
		_, message, err := user_connection.ws.ReadMessage()
		if err != nil {
			break
		}
		user_connection.UserId, _ = strconv.Atoi(string(message))
		log.Print(string(message))
		user_connection.ListenNotification()
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	user_connection := &UserConnection{ws: ws, ReadyForGetNotification: false}
	go user_connection.Listen()
}

func serveMain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	homeTempl.Execute(w, r.Host)
}

func main() {
	flag.Parse()
	http.HandleFunc("/", serveMain)
	http.HandleFunc("/ws", serveWs)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
