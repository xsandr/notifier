package main

import (
	"testing"

	"github.com/streadway/amqp"
)

func TestMessageParsing(t *testing.T) {
	headers := make(amqp.Table, 0)
	var ttl int32 = 13
	headers["ttl"] = ttl
	body := "test message"
	message := amqp.Delivery{
		RoutingKey: "user.18",
		Body:       []byte(body),
		Headers:    headers,
	}
	parsed, _ := ParseMessage(message)
	if parsed.Message != body || parsed.UID != 18 || parsed.TTL != 13 {
		t.Fatalf("%v", parsed)
	}

}
