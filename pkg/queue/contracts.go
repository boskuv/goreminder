package queue

import "context"

// Publisher defines the minimal interface required for publishing messages
// to a queue. It is implemented by Producer and can be implemented by
// alternative or no-op publishers.
type Publisher interface {
	Publish(ctx context.Context, message interface{}) error
}

// NoopPublisher is a Publisher implementation that discards all messages.
// It is useful for running the application in "DB-only" mode without
// requiring a message queue.
type NoopPublisher struct{}

func (NoopPublisher) Publish(ctx context.Context, message interface{}) error {
	return nil
}

