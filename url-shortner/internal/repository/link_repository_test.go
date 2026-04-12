package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kernelshard/url-shortner-platform/internal/model"
	"github.com/stretchr/testify/require"
)

// TestClaimPendingOutboxTx_Concurrent tests the concurrent claim of pending
// outbox transactions.
func TestClaimPendingOutboxTx_Concurrent(t *testing.T) {
	ctx := context.Background()

	repo, db := setupTestRepo(t)
	cleanDB(t, db)

	// insert one event
	event := model.OutBoxEvent{
		ID:          uuid.New(),
		EventType:   "test",
		Payload:     []byte(`{}`),
		CreatedAt:   time.Now().UTC(),
		NextRetryAt: time.Now().UTC().Add(-10 * time.Second),
		Processed:   false,
		RetryCount:  0,
		ClaimedAt:   nil,
	}

	err := repo.InsertOutbox(ctx, event)
	require.NoError(t, err)

	var wg sync.WaitGroup
	results := make(chan int, 2)

	// Test concurrent claim of pending outbox transactions
	worker := func() {
		defer wg.Done()

		err := repo.WithTx(ctx, func(tx pgx.Tx) error {
			// it locks the row so other transactions cannot claim it
			// the event should only be claimed by one worker
			events, err := repo.ClaimPendingOutboxTx(ctx, tx, 1)
			if err != nil {
				return err
			}

			results <- len(events)
			return nil
		})

		require.NoError(t, err)
	}

	wg.Add(2)
	go worker()
	go worker()
	wg.Wait()
	close(results)

	// verify that both workers claimed one event as it was locked by the first worker
	total := 0
	for r := range results {
		total += r
	}
	// each worker should claim 1 event
	require.Equal(t, 1, total)

}

// TestClaimPendingOutboxTx_NotRefetched verifies that a claimed event is not refetched by another worker
// - cause on first query claimed_at is not nil any more
func TestClaimPendingOutboxTx_NotRefetched(t *testing.T) {
	ctx := context.Background()

	repo, db := setupTestRepo(t)
	cleanDB(t, db)

	event := model.OutBoxEvent{
		ID:          uuid.New(),
		EventType:   "test",
		Payload:     []byte(`{}`),
		CreatedAt:   time.Now().UTC(),
		NextRetryAt: time.Now().UTC().Add(-10 * time.Second),
		Processed:   false,
	}

	require.NoError(t, repo.InsertOutbox(ctx, event))

	err := repo.WithTx(ctx, func(tx pgx.Tx) error {
		events, err := repo.ClaimPendingOutboxTx(ctx, tx, 1)
		require.NoError(t, err)
		require.Len(t, events, 1)

		return nil
	})
	require.NoError(t, err)

	// second claim should not fetch the same event as claimed_at is not nil any more
	err = repo.WithTx(ctx, func(tx pgx.Tx) error {
		events, err := repo.ClaimPendingOutboxTx(ctx, tx, 1)
		require.NoError(t, err)
		require.Len(t, events, 0)
		return nil
	})

	require.NoError(t, err)

}
