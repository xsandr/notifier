package main

import (
	"flag"
	"log"
	"net/http"
	"regexp"
	"text/template"

	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", ":5000", "ws service address")
	rabbit   = flag.String("rabbit", "amqp://guest:guest@localhost:5672/", "AMQP URI")
	exchange = flag.String("exchange", "notifications", "Durable, non-auto-deleted AMQP exchange name")
	queue    = flag.String("queue", "notifications", "Queue name")
	routing  = flag.String("routing key", "user.*", "Routing key for queue")
	ttl      = flag.Int64("ttl", 3*86400000, "default TTL for undelivered message (msec)")

	certFile = flag.String("cert", "", "Cert for TLS")
	keyFile  = flag.String("keyfile", "", "Key for TLS")

	re           = regexp.MustCompile("user.(\\d+)")
	registry     = NewRegistry()
	isTTLEnabled bool

	homeTempl = template.Must(template.ParseFiles("index.html"))
	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	NewUserConnection(ws).Listen()
}

func serveMain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	homeTempl.Execute(w, r.Host)
}

func main() {
	flag.Parse()
	isTTLEnabled = *ttl > 0
	go registry.ListenAndSendMessages()

	http.HandleFunc("/", serveMain)
	http.HandleFunc("/ws", serveWs)
	log.Print("Server started")
	if *certFile != "" && *keyFile != "" {
		log.Fatal(http.ListenAndServeTLS(*addr, *certFile, *keyFile, nil))
	} else {
		log.Fatal(http.ListenAndServe(*addr, nil))
	}
}
