package queue

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type ProducerConfig struct {
	Host      string // TODO: case
	Port      string
	User      string
	Password  string
	QueueName string
	Exchange  string
}

type Producer struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queue      string
	exchange   string
}

func NewProducer(cfg *ProducerConfig) (*Producer, error) {
	rabbitMQURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		cfg.User, cfg.Password, cfg.Host, cfg.Port,
	)

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ channel: %v", err)
	}

	// TODO: если очередь уже существует?
	_, err = ch.QueueDeclare(cfg.QueueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	return &Producer{
		connection: conn,
		channel:    ch,
		queue:      cfg.QueueName,
		exchange:   cfg.Exchange,
	}, nil
}

// Implement a Close method
func (p *Producer) Close() {
	p.channel.Close()
	p.connection.Close()
}

func (p *Producer) Publish(message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "failed to marshal message")
	}

	err = p.channel.Publish(
		p.exchange, p.queue, false, false,
		amqp.Publishing{ContentType: "application/json", Body: body},
	)

	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return errors.Wrap(err, "failed to publish message")
	}

	log.Println("Message published successfully")
	return nil
}
