package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kernelshard/url-shortner-platform/internal/event"
	"github.com/kernelshard/url-shortner-platform/internal/model"
	"github.com/kernelshard/url-shortner-platform/internal/repository"
	"github.com/stretchr/testify/assert"
)

type mockPublisher struct {
	shouldFail bool
}

func (m *mockPublisher) Publish(ctx context.Context, e event.Event) error {
	if m.shouldFail {
		return errors.New("publish failed")
	}
	return nil
}

// TestRetryScheduledOnPublishFailure tests that the outbox worker retries a failed event after a delay.
func TestRetryScheduledOnPublishFailure(t *testing.T) {
	repo, db := repository.SetupTestRepo(t)
	pgRepo := repo.(*repository.PostgresLinkRepository)
	defer repository.CleanDB(t, db)
	pub := &mockPublisher{shouldFail: true}

	ctx := context.Background()

	// insert one outbox event
	e := model.OutBoxEvent{
		ID:          uuid.New(),
		EventType:   "test.event",
		Payload:     []byte(`{}`),
		Processed:   false,
		NextRetryAt: time.Now().UTC().Add(-time.Minute), // must be in the past to trigger retry
		RetryCount:  0,
	}

	// insert the outbox event into the database for testing
	repository.InsertBoxForTest(t, pgRepo, e)

	// run one batch
	processBatch(repo, pub)

	// fetch updatedEvent
	updated, err := getOutboxEvent(ctx, t, repo, e.ID)
	assert.NoError(t, err)

	// assertions
	assert.False(t, updated.Processed, "event should NOT be marked processed on failure")
	assert.Equal(t, 1, updated.RetryCount, "retry count should increase on failure")
	assert.True(t, updated.NextRetryAt.After(time.Now().UTC()), "next retry time should be in the future")

}

// getOutboxEvent retrieves an outbox event by ID from the database.
func getOutboxEvent(ctx context.Context, t *testing.T, repo repository.LinkOutboxRepository, id uuid.UUID) (model.OutBoxEvent, error) {
	t.Helper()
	pgRepo := repo.(*repository.PostgresLinkRepository)
	query := `
		SELECT id, event_type, payload, processed, next_retry_at, retry_count, created_at, claimed_at
		FROM outbox_events
		WHERE id = $1
	`

	var e model.OutBoxEvent
	err := pgRepo.DB().QueryRow(ctx, query, id).Scan(
		&e.ID,
		&e.EventType,
		&e.Payload,
		&e.Processed,
		&e.NextRetryAt,
		&e.RetryCount,
		&e.CreatedAt,
		&e.ClaimedAt,
	)
	return e, err
}
