package event

import "context"

// Publisher is an interface for publishing events.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}
