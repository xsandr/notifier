package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/streadway/amqp"
)

// Consumer is listening RabbitMQ and send messages to Registry
type Consumer struct {
	Messages            chan Message
	UndeliveredMessages chan Message

	connection *amqp.Connection
}

// NewConsumer is constructor for Consumer
func NewConsumer() *Consumer {
	return &Consumer{Messages: make(chan Message), UndeliveredMessages: make(chan Message)}
}

// Run starts two goroutine
// first is listening RabbitMQ for new messages
// second - publishes messages for offline users
func (c *Consumer) Run() {
	conn, err := amqp.Dial(*rabbit)
	if err != nil {
		panic(err)
	}
	c.connection = conn

	go c.GetMessages()
	go c.publishMessages()
}

// GetMessages listens RabbitMQ and send new messages into Messages channel
func (c *Consumer) GetMessages() {
	channel := c.GetChannel()
	defer channel.Close()

	err := channel.ExchangeDeclare(*exchange, "topic", false, false, false, false, nil)
	checkError(err)
	for message := range c.GetDeliveries(*queue, *routing, channel) {
		parsedMessage, err := ParseMessage(message)
		if err != nil {
			log.Println(err)
			continue
		}
		c.Messages <- *parsedMessage
	}
}

// PublishUndeliveredMessage publishes for offline users
func (c *Consumer) PublishUndeliveredMessage(message Message) {
	c.UndeliveredMessages <- message
}

// publishMessages publishes messages into queue undelivered.user.<UID>
func (c *Consumer) publishMessages() {
	channel := c.GetChannel()
	defer channel.Close()
	err := channel.ExchangeDeclare(*exchange, "topic", false, false, false, false, nil)
	checkError(err)

	for message := range c.UndeliveredMessages {
		log.Printf("Get undelivered message for user %d: %s", message.UID, message.Message)
		key := fmt.Sprintf("undelivered.user.%d", message.UID)
		_, err = channel.QueueDeclare(key, false, false, false, false, nil)
		checkError(err)
		err = channel.QueueBind(key, key, *exchange, false, nil)
		checkError(err)

		channel.Publish(*exchange, key, false, false, amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "text/plain",
			ContentEncoding: "",
			Expiration:      fmt.Sprintf("%d", message.TTL),
			Body:            []byte(message.Message),
			DeliveryMode:    amqp.Transient,
			Priority:        0,
		})
	}
}

// GetChannel returns new AMQP channel
func (c *Consumer) GetChannel() *amqp.Channel {
	channel, err := c.connection.Channel()
	if err != nil {
		panic(err)
	}
	return channel
}

func checkError(err error) {
	if err != nil {
		log.Println("Check your RabbitMQ connection!")
	}
}

// GetUndeliveredMessage get new AMQP channel and gets all undelivered messages for UID
func (c *Consumer) GetUndeliveredMessage(uid int) {
	go func(uid int) {
		qname := fmt.Sprintf("undelivered.user.%d", uid)
		channel := c.GetChannel()
		defer channel.Close()
		ticker := time.NewTicker(5 * time.Second)

		for {
			select {
			case message := <-c.GetDeliveries(qname, qname, channel):
				parsedMessage, err := ParseMessage(message)
				if err != nil {
					log.Println(err)
					continue
				}
				c.Messages <- *parsedMessage
			case <-ticker.C:
				queue, err := channel.QueueInspect(qname)
				if err != nil || queue.Messages == 0 {
					channel.Close()
					return
				}
			}
		}
	}(uid)
}

// GetDeliveries returns channel with messages from RabbitMQ, for particular queue
func (c *Consumer) GetDeliveries(qname, routing string, channel *amqp.Channel) <-chan amqp.Delivery {
	queue, err := channel.QueueDeclare(qname, false, false, false, false, nil)
	checkError(err)

	err = channel.QueueBind(queue.Name, routing, *exchange, false, nil)
	checkError(err)

	deliveries, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	checkError(err)
	return deliveries
}

// ParseMessage parses amqp.Delivery and returns Message instance
func ParseMessage(message amqp.Delivery) (*Message, error) {
	matches := re.FindStringSubmatch(message.RoutingKey)
	if len(matches) == 0 {
		return nil, errors.New("Unknown routing key")
	}
	uid, err := strconv.Atoi(matches[len(matches)-1])
	if err != nil {
		return nil, err
	}
	var messageTTL int64
	if ttlHeader, ok := message.Headers["ttl"]; ok && ttlHeader != nil {
		if ttlInt, ok := ttlHeader.(int32); ok {
			messageTTL = int64(ttlInt)
		}
	}
	return &Message{uid, string(message.Body), messageTTL}, nil
}
