package event

import (
	"context"
	"log"
)

type InMemoryPublisher struct {
}

func NewInMemoryPublisher() *InMemoryPublisher {
	return &InMemoryPublisher{}
}

func (p *InMemoryPublisher) Publish(ctx context.Context, event Event) error {
	log.Printf("publishing event: %+v\n", event)
	return nil
}
