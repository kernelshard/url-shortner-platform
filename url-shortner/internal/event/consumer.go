package event

import (
	"context"
	"log"
)

// Consumer handles events.
// It receives events and performs actions based on their type and data.
type Consumer interface {
	Handle(ctx context.Context, e Event) error
}

type LoggingConsumer struct{}

func NewLoggingConsumer() *LoggingConsumer {
	return &LoggingConsumer{}
}

// Handle logs the event type and data.
func (c *LoggingConsumer) Handle(ctx context.Context, e Event) error {
	log.Printf("event receiver: type=%s, data=%v", e.Type, e.Data)
	return nil
}
