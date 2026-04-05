package event

import (
	"context"
)

// NoOpPublisher implements the Publisher interface but does nothing.
// Useful for tests or when event delivery is disabled.
type NoOpPublisher struct{}

// NewNoOpPublisher creates a new NoOpPublisher.
func NewNoOpPublisher() *NoOpPublisher {
	return &NoOpPublisher{}
}

// Publish is a no-op.
func (p *NoOpPublisher) Publish(ctx context.Context, event Event) error {
	return nil
}
