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
	host                 string
	port                 string
	user                 string
	password             string
	queueName            string
	exchange             string
	connectionRetries    int
	connectionRetryDelay time.Duration
}

// NewProducerConfig creates a new ProducerConfig with the provided parameters
func NewProducerConfig(host, port, user, password, queueName, exchange string, connectionRetries int, connectionRetryDelay time.Duration) *ProducerConfig {
	return &ProducerConfig{
		host:                 host,
		port:                 port,
		user:                 user,
		password:             password,
		queueName:            queueName,
		exchange:             exchange,
		connectionRetries:    connectionRetries,
		connectionRetryDelay: connectionRetryDelay * time.Second,
	}
}

type Producer struct {
	ctx        context.Context
	connection *rabbitroutine.Connector
	channel    *amqp091.Channel
	queue      string
	exchange   string
	publisher  *rabbitroutine.RetryPublisher
}

// NewProducer initializes a new Producer with the given configuration
func NewProducer(cfg *ProducerConfig) (*Producer, error) {
	ctx := context.Background()

	rabbitMQURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		cfg.user, cfg.password, cfg.host, cfg.port,
	)

	conn := rabbitroutine.NewConnector(rabbitroutine.Config{
		ReconnectAttempts: uint(cfg.connectionRetries),
		Wait:              cfg.connectionRetryDelay,
	})

	// try to connect to RabbitMQ
	go func() {
		err := conn.Dial(ctx, rabbitMQURL)
		if err != nil {
			log.Fatalf("failed to connect to RabbitMQ: %v", err)
		}
	}()

	pool := rabbitroutine.NewPool(conn)
	ensurePub := rabbitroutine.NewEnsurePublisher(pool)
	publisher := rabbitroutine.NewRetryPublisher(
		ensurePub,
		rabbitroutine.PublishMaxAttemptsSetup(16),
		rabbitroutine.PublishDelaySetup(rabbitroutine.LinearDelay(10*time.Millisecond)),
	)

	ch, err := conn.Channel(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create RabbitMQ channel")
	}

	if cfg.exchange != "" {
		err = ch.ExchangeDeclare(cfg.exchange, "direct", true, false, false, false, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to declare exchange")
		}
	}

	_, err = ch.QueueDeclare(cfg.queueName, true, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to declare queue")
	}

	producer := &Producer{
		ctx:        ctx,
		connection: conn,
		channel:    ch,
		queue:      cfg.queueName,
		exchange:   cfg.exchange,
		publisher:  publisher,
	}

	return producer, nil
}

// Close closes the RabbitMQ channel and connection
func (p *Producer) Close() error {
	err := p.channel.Close()

	if err != nil {
		return errors.Wrap(err, "failed to close channel")
	}

	return nil
}

// Publish sends a message to the RabbitMQ queue
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

	defer cancel()

	if err != nil {
		return errors.Wrap(err, "failed to publish message")
	}

	return nil
}
