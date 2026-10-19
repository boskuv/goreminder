package queue

import "context"

// Publisher defines the minimal interface required for publishing messages
// to a queue. It is implemented by Producer and can be implemented by
// alternative or no-op publishers.
type Publisher interface {
	Publish(ctx context.Context, message interface{}) error
}

// HealthChecker reports whether the queue broker is reachable.
// Implemented by Producer; NoopPublisher is always healthy.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// NoopPublisher is a Publisher implementation that discards all messages.
// It is useful for running the application in "DB-only" mode without
// requiring a message queue.
type NoopPublisher struct{}

func (NoopPublisher) Publish(_ context.Context, _ interface{}) error {
	return nil
}

func (NoopPublisher) Ping(_ context.Context) error {
	return nil
}

