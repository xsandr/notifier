package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/streadway/amqp"
)

type Consumer struct {
	Messages            chan Message
	UndeliveredMessages chan Message

	connection *amqp.Connection
}

func NewConsumer() *Consumer {
	return &Consumer{Messages: make(chan Message), UndeliveredMessages: make(chan Message)}
}

func (c *Consumer) Run() {
	conn, err := amqp.Dial(*rabbit)
	if err != nil {
		panic(err)
	}
	c.connection = conn

	go c.GetMessages()
	go c.publish_messages()
}

func (c *Consumer) GetMessages() {
	channel := c.GetChannel()
	defer channel.Close()

	err := channel.ExchangeDeclare(*exchange, "topic", false, false, false, false, nil)
	check_error(err)
	for message := range c.GetDeliveries(*queue, *routing, channel) {
		parsed_message, err := ParseMessage(message)
		if err != nil {
			log.Println(err)
			continue
		}
		c.Messages <- *parsed_message
	}
}

func (c *Consumer) PublishUndeliveredMessage(message Message) {
	c.UndeliveredMessages <- message
}

func (c *Consumer) publish_messages() {
	channel := c.GetChannel()
	defer channel.Close()
	err := channel.ExchangeDeclare(*exchange, "topic", false, false, false, false, nil)
	check_error(err)

	for message := range c.UndeliveredMessages {
		log.Printf("Get undelivered message for user %d: %s", message.UID, message.Message)
		key := fmt.Sprintf("undelivered.user.%d", message.UID)
		_, err = channel.QueueDeclare(key, false, false, false, false, nil)
		check_error(err)
		err = channel.QueueBind(key, key, *exchange, false, nil)
		check_error(err)

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

func (c *Consumer) GetChannel() *amqp.Channel {
	channel, err := c.connection.Channel()
	if err != nil {
		panic(err)
	}
	return channel
}

func check_error(err error) {
	if err != nil {
		log.Println("Check your RabbitMQ connection!")
	}
}

func (c *Consumer) GetUndeliveredMessage(uid int) {
	go func(uid int) {
		qname := fmt.Sprintf("undelivered.user.%d", uid)
		channel := c.GetChannel()
		defer channel.Close()
		ticker := time.NewTicker(5 * time.Second)

		for {
			select {
			case message := <-c.GetDeliveries(qname, qname, channel):
				parsed_message, err := ParseMessage(message)
				if err != nil {
					log.Println(err)
					continue
				}
				c.Messages <- *parsed_message
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

func (c *Consumer) GetDeliveries(qname, routing string, channel *amqp.Channel) <-chan amqp.Delivery {
	queue, err := channel.QueueDeclare(qname, false, false, false, false, nil)
	check_error(err)

	err = channel.QueueBind(queue.Name, routing, *exchange, false, nil)
	check_error(err)

	deliveries, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	check_error(err)
	return deliveries
}

func ParseMessage(message amqp.Delivery) (*Message, error) {
	matches := re.FindStringSubmatch(message.RoutingKey)
	if len(matches) == 0 {
		return nil, errors.New("Unknown routing key")
	}
	uid, err := strconv.Atoi(matches[len(matches)-1])
	if err != nil {
		return nil, err
	}
	var message_ttl int64 = 0
	if ttl_header, ok := message.Headers["ttl"]; ok && ttl_header != nil {
		if ttl_int, ok := ttl_header.(int32); ok {
			message_ttl = int64(ttl_int)
		}
	}
	return &Message{uid, string(message.Body), message_ttl}, nil
}
