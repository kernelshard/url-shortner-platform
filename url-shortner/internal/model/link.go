package model

import (
	"time"

	"github.com/google/uuid"
)

// Link represents a shortened URL record with metadata.
// It contains the original URL, a short code for reference,
// and timestamp information for creation and expiration.
type Link struct {
	ID          uuid.UUID
	OriginalURL string
	ShortCode   string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}
