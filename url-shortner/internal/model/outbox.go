package model

import (
	"time"

	"github.com/google/uuid"
)

// OutBoxEvent represents an event to be processed by the outbox pattern.
type OutBoxEvent struct {
	ID        uuid.UUID
	Type      string
	Payload   []byte
	Processed bool
	CreatedAt time.Time
}
