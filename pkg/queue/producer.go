package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/furdarius/rabbitroutine"
	"github.com/pkg/errors"
	"github.com/rabbitmq/amqp091-go"
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
	ctx        context.Context
	connection *rabbitroutine.Connector
	channel    *amqp091.Channel
	queue      string
	exchange   string
	publisher  *rabbitroutine.RetryPublisher
}

func NewProducer(cfg *ProducerConfig) (*Producer, error) {
	ctx := context.Background()

	rabbitMQURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		cfg.User, cfg.Password, cfg.Host, cfg.Port,
	)

	conn := rabbitroutine.NewConnector(rabbitroutine.Config{
		ReconnectAttempts: 20,
		Wait:              10 * time.Second,
	})

	pool := rabbitroutine.NewPool(conn)
	ensurePub := rabbitroutine.NewEnsurePublisher(pool)
	publisher := rabbitroutine.NewRetryPublisher(
		ensurePub,
		rabbitroutine.PublishMaxAttemptsSetup(16),
		rabbitroutine.PublishDelaySetup(rabbitroutine.LinearDelay(10*time.Millisecond)),
	)

	go func() {
		err := conn.Dial(ctx, rabbitMQURL)
		if err != nil {
			log.Fatalf("failed to connect to RabbitMQ: %v", err)
			//return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
		}
	}()

	ch, err := conn.Channel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ channel: %v", err)
	}

	if cfg.Exchange != "" {
		err = ch.ExchangeDeclare(cfg.Exchange, "direct", true, false, false, false, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to declare exchange: %v", err)
		}

		// TODO
		// err = ch.QueueBind(queueName, queueName, exchangeName, false, nil)
		// if err != nil {
		// 	log.Fatalf("failed to bind queue: %v", err)
		// }
	}

	// TODO: если очередь уже существует?
	_, err = ch.QueueDeclare(cfg.QueueName, true, false, false, false, nil)
	// TODO: set right options
	// 	name,
	// 	false, // Durable
	// 	false, // Delete when unused
	// 	false, // Exclusive
	// 	false, // No-wait
	// 	nil,   // Arguments
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	producer := &Producer{
		ctx:        ctx,
		connection: conn,
		channel:    ch,
		queue:      cfg.QueueName,
		exchange:   cfg.Exchange,
		publisher:  publisher,
	}

	return producer, nil
}

func (p *Producer) Close() error {
	err := p.channel.Close()

	if err != nil {
		return errors.Wrap(err, "failed to close channel")
	}

	// TODO: graceful close connection
	// err = p.connection.Close()
	// if err != nil {
	// 	return errors.Wrap(err, "failed to close connection")
	// }

	return nil
}

func (p *Producer) Publish(message interface{}) error {

	body, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "failed to marshal message")
	}

	timeoutCtx, cancel := context.WithTimeout(p.ctx, 100*time.Millisecond)
	err = p.publisher.Publish(
		timeoutCtx, p.exchange, p.queue,
		amqp091.Publishing{ContentType: "application/json", Body: body},
	)

	cancel()

	if err != nil {
		//log.Printf("failed to publish message: %v", err)
		return errors.Wrap(err, "failed to publish message")
	}

	log.Println("Message published successfully") // TODO: is published + logging

	return nil
}
