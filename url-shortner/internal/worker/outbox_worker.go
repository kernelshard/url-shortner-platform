package worker

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/kernelshard/url-shortner-platform/internal/event"
	"github.com/kernelshard/url-shortner-platform/internal/model"
	"github.com/kernelshard/url-shortner-platform/internal/repository"
)

// StartOutboxWorker starts the outbox worker that processes pending outbox events.
// - Claims pending outbox events using a transaction.
// - Publishes the events to the event bus.
// - Marks the events as processed using a transaction.
func StartOutboxWorker(repo repository.LinkOutboxRepository, pub event.Publisher) {
	log.Printf("outbox: worker started")
	for {
		processBatch(repo, pub)
		time.Sleep(3 * time.Second)
	}
}

func processBatch(repo repository.LinkOutboxRepository, pub event.Publisher) {

	ctx := context.Background()

	var events []model.OutBoxEvent
	err := repo.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		events, err = repo.ClaimPendingOutboxTx(ctx, tx, 10)
		return err
	})

	if err != nil {
		log.Printf("outbox: claim failed: %v", err)
		return
	}

	log.Printf("outbox: publishing %d events", len(events))
	for _, e := range events {
		log.Printf("outbox: publishing event: %s", e.ID)

		err := pub.Publish(ctx, event.Event{
			EventID: e.ID.String(),
			Type:    e.EventType,
			Data:    e.Payload,
		})

		if err != nil {
			log.Printf("outbox: publish failed id=%s err=%v", e.ID, err)
			const maxRetries = 10

			if e.RetryCount+1 >= maxRetries {
				err = repo.WithTx(ctx, func(tx pgx.Tx) error {
					return repo.MarkOutboxDeadTx(ctx, tx, e.ID)
				})
				if err != nil {
					log.Printf("outbox: mark dead failed id=%s err=%v", e.ID, err)
				}
				continue
			}

			nextRetry := computeNextRetry(e.RetryCount + 1)

			err := repo.WithTx(ctx, func(tx pgx.Tx) error {
				return repo.UpdateRetryStateTx(ctx, tx, e.ID, nextRetry)
			})
			if err != nil {
				log.Printf("outbox: update retry state failed id=%s err=%v", e.ID, err)
			}
			log.Printf("outbox: retry scheduled id=%s at=%s", e.ID, nextRetry)
			continue
		}

		err = repo.WithTx(ctx, func(tx pgx.Tx) error {
			return repo.MarkOutboxProcessedTx(ctx, tx, e.ID)
		})
		if err != nil {
			log.Printf("outbox: mark processed failed id=%s err=%v", e.ID, err)
			continue
		}
		log.Printf("outbox: event processed success id=%s", e.ID)
	}
}

// computeNextRetry computes the next retry time with exponential backoff and jitter
func computeNextRetry(retryCount int) time.Time {
	base := 5 * time.Second
	max := 5 * time.Minute           // without max cap it would grow exponentially
	retryCount = min(retryCount, 10) // cap retry count at 10 to avoid exponential growth

	// exponential backoff with jitter
	delay := base * time.Duration(1<<retryCount)
	delay = min(delay, max)

	jitter := time.Duration(rand.Int63n(int64(delay / 2)))

	// suppose 1000 events failed in a row
	// jitter will spread out the retries over time, so they don't all happen at once
	nextRetry := time.Now().Add(delay + jitter)
	return nextRetry
}
