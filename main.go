package main

import (
	"expvar"
	"flag"
	"log"
	"net/http"
	"regexp"
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
	usersCount       = expvar.NewInt("active users")
	connectionsCount = expvar.NewInt("active connections")
)

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
	log.Print("Server started")
	if *ssl {
		http.ListenAndServeTLS(*addr, *certFile, *keyFile, nil)
	} else {
		http.ListenAndServe(*addr, nil)
	}
}
