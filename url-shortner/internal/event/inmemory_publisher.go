package event

import (
	"context"
	"log"
)

// InMemoryPublisher implements the Publisher interface and publishes events in-memory.
type InMemoryPublisher struct {
	consumer Consumer
}

func NewInMemoryPublisher(consumer Consumer) *InMemoryPublisher {
	return &InMemoryPublisher{consumer: consumer}
}

func (p *InMemoryPublisher) Publish(ctx context.Context, event Event) error {
	log.Printf("publishing event: %+v\n", event)
	if p.consumer != nil {
		return p.consumer.Handle(ctx, event)
	}
	return nil
}
