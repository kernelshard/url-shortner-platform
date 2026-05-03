package service

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/repository"
	"github.com/stretchr/testify/require"
)

// Test_ProcessLinkCreate_Idempotent verifies that the ProcessLinkCreated method is idempotent.
// It will process the same event multiple times and verify that only one row is created in the database.

func Test_ProcessLinkCreate_Idempotent(t *testing.T) {
	db := repository.TestDB(t)

	repo := repository.NewPostgresProcessedEventRepository(db)
	svc := NewEventService(repo)

	eventID := uuid.New()
	email := "test@example.com"

	var wg sync.WaitGroup
	concurrency := 50
	// process same event multiple times but only one row should be created in the database
	for range concurrency {
		wg.Go(func() {
			_ = svc.ProcessLinkCreated(context.Background(), eventID, email)
		})
	}

	wg.Wait()

	// verify only one row exists
	var count int

	err := db.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM processed_events WHERE event_id = $1`,
		eventID,
	).Scan(&count)

	require.NoError(t, err)
	require.Equal(t, 1, count) // only one row should be created, duplicate  should be ignored

	// verify email_deliveries only one too, it's in same transaction

	err = db.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM email_deliveries WHERE event_id = $1`,
		eventID,
	).Scan(&count)

	require.NoError(t, err)
	require.Equal(t, 1, count) // only one row should be created, duplicate  should be ignored

}
