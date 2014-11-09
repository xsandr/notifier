package main

import "github.com/streadway/amqp"

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
