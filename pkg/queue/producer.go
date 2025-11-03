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
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/boskuv/goreminder/pkg/logger"
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
	cancel     context.CancelFunc
	connection *rabbitroutine.Connector
	channel    *amqp091.Channel
	queue      string
	exchange   string
	publisher  *rabbitroutine.RetryPublisher
	tracer     trace.Tracer
	logger     zerolog.Logger
}

// NewProducer initializes a new Producer with the given configuration
func NewProducer(cfg *ProducerConfig, logger zerolog.Logger) (*Producer, error) {
	ctx, cancel := context.WithCancel(context.Background())

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
		for {
			if ctx.Err() != nil {
				return
			}

			err := conn.Dial(ctx, rabbitMQURL)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Failed to connect to RabbitMQ: %v, retrying...", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(cfg.connectionRetryDelay):
				}
				continue
			}

			// If we get here, connection was successful
			log.Println("Connected to RabbitMQ")

			// Wait for context cancellation or connection drop
			<-ctx.Done()
			return
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
		cancel()
		return nil, errors.Wrap(err, "failed to create RabbitMQ channel")
	}

	if cfg.exchange != "" {
		err = ch.ExchangeDeclare(cfg.exchange, "direct", true, false, false, false, nil)
		if err != nil {
			cancel()
			return nil, errors.Wrap(err, "failed to declare exchange")
		}
	}

	_, err = ch.QueueDeclare(cfg.queueName, true, false, false, false, nil)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to declare queue")
	}

	producer := &Producer{
		ctx:        ctx,
		cancel:     cancel,
		connection: conn,
		channel:    ch,
		queue:      cfg.queueName,
		exchange:   cfg.exchange,
		publisher:  publisher,
		tracer:     otel.Tracer("queue-producer"),
		logger:     logger,
	}

	return producer, nil
}

// Close closes the RabbitMQ channel and cancels background routines
func (p *Producer) Close() error {
	// cancel background goroutines
	if p.cancel != nil {
		p.cancel()
	}
	// close channel
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			// ignore already-closed errors
			if !errors.Is(err, amqp091.ErrClosed) {
				return errors.Wrap(err, "failed to close channel")
			}
		}
	}
	return nil
}

// Publish sends a message to the RabbitMQ queue
func (p *Producer) Publish(ctx context.Context, message interface{}) error {
	ctx, span := p.tracer.Start(ctx, "queue_producer.Publish",
		trace.WithAttributes(
			attribute.String("queue.name", p.queue),
			attribute.String("exchange.name", p.exchange),
		))
	defer span.End()

	log := logger.WithTraceContext(ctx, p.logger)

	// Extract task name from message if it's a map
	taskName := ""
	if msgMap, ok := message.(map[string]interface{}); ok {
		if task, ok := msgMap["task"].(string); ok {
			taskName = task
			span.SetAttributes(attribute.String("task.name", task))
		}
	}

	log.Debug().
		Str("queue.name", p.queue).
		Str("exchange.name", p.exchange).
		Str("task.name", taskName).
		Msg("publishing message to queue")

	body, err := json.Marshal(message)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("failed to marshal message")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to marshal message")
	}

	span.SetAttributes(attribute.Int("message.size", len(body)))
	log.Debug().
		Int("message.size", len(body)).
		Str("queue.name", p.queue).
		Msg("message marshaled, publishing to queue")

	// Ensure bounded publish time; prefer request ctx if provided
	var timeoutCtx context.Context
	var cancel context.CancelFunc
	if ctx != nil {
		timeoutCtx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	} else {
		timeoutCtx, cancel = context.WithTimeout(p.ctx, 100*time.Millisecond)
	}
	defer cancel()

	err = p.publisher.Publish(
		timeoutCtx, p.exchange, p.queue,
		amqp091.Publishing{ContentType: "application/json", Body: body},
	)

	if err != nil {
		log.Debug().
			Err(err).
			Str("queue.name", p.queue).
			Msg("failed to publish message to queue")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to publish message")
	}

	log.Debug().
		Str("queue.name", p.queue).
		Int("message.size", len(body)).
		Msg("message published successfully")
	span.SetStatus(codes.Ok, "message published successfully")
	return nil
}
