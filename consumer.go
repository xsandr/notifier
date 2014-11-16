package main

import (
	"fmt"

	"github.com/streadway/amqp"
)

func check_error(err error) {
	if err != nil {
		panic(err)
	}
}

func GetMessagesChannel() <-chan amqp.Delivery {
	connection, err := amqp.Dial(*rabbit)
	check_error(err)
	channel, err := connection.Channel()
	check_error(err)
	err = channel.ExchangeDeclare(
		*exchange, // name of the exchange
		"topic",   // type
		false,     // durable
		false,     // delete when complete
		false,     // internal
		false,     // noWait
		nil,       // arguments
	)
	check_error(err)
	queue, err := channel.QueueDeclare(
		*queue, // name of the queue
		false,  // durable
		false,  // delete when usused
		false,  // exclusive
		false,  // noWait
		nil,    // arguments
	)
	check_error(err)
	err = channel.QueueBind(
		queue.Name, // name of the queue
		*routing,   // bindingKey
		*exchange,  // sourceExchange
		false,      // noWait
		nil,        // arguments
	)
	check_error(err)

	deliveries, err := channel.Consume(
		queue.Name, // name
		"",         // consumerTag,
		false,      // noAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)
	check_error(err)
	return deliveries
}

func PublishUndeliveredMessage(user_id int, message []byte, ttl int64) {
	connection, err := amqp.Dial(*rabbit)
	defer connection.Close()
	check_error(err)
	channel, err := connection.Channel()
	check_error(err)
	key := fmt.Sprintf("undelivered.user.%d", user_id)
	err = channel.ExchangeDeclare(
		*exchange,
		"topic", // type
		false,   // durable
		false,   // delete when complete
		false,   // internal
		false,   // noWait
		nil,     // arguments
	)
	check_error(err)
	_, err = channel.QueueDeclare(
		key,
		false, // durable
		false, // delete when usused
		false, // exclusive
		false, // noWait
		nil,   // arguments
	)
	check_error(err)
	err = channel.QueueBind(
		key,       // name of the queue
		key,       // bindingKey
		*exchange, // sourceExchange
		false,     // noWait
		nil,       // arguments
	)
	check_error(err)
	channel.Publish(*exchange, key, false, false, amqp.Publishing{
		Headers:         amqp.Table{},
		ContentType:     "text/plain",
		ContentEncoding: "",
		Expiration:      fmt.Sprintf("%d", ttl),
		Body:            message,
		DeliveryMode:    amqp.Transient,
		Priority:        0,
	})
}

func GetUndeliveredMessage(user_id int) (<-chan amqp.Delivery, *amqp.Connection) {
	connection, err := amqp.Dial(*rabbit)
	check_error(err)
	channel, err := connection.Channel()
	check_error(err)

	key := fmt.Sprintf("undelivered.user.%d", user_id)
	queue, err := channel.QueueDeclare(
		key,
		false, // durable
		false, // delete when usused
		false, // exclusive
		false, // noWait
		nil,   // arguments
	)
	check_error(err)
	err = channel.QueueBind(
		queue.Name, // name of the queue
		key,        // bindingKey
		*exchange,  // sourceExchange
		false,      // noWait
		nil,        // arguments
	)
	check_error(err)

	deliveries, err := channel.Consume(
		queue.Name, // name
		"",         // consumerTag,
		false,      // noAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)
	check_error(err)
	return deliveries, connection
}
