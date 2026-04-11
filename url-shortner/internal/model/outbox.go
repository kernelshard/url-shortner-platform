package model

import (
	"time"

	"github.com/google/uuid"
)

// OutBoxEvent represents an event to be processed by the outbox pattern.
type OutBoxEvent struct {
	ID          uuid.UUID `db:"id"`
	EventType   string    `db:"event_type"`
	Payload     []byte    `db:"payload"`
	Processed   bool      `db:"processed"`
	NextRetryAt time.Time `db:"next_retry_at"`
	RetryCount  int       `db:"retry_count"`
	CreatedAt   time.Time `db:"created_at"`
}
