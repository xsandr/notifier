package main

type Message struct {
	UID               int
	Message           string
	TTL               int64
	AllUserConnection bool
}
