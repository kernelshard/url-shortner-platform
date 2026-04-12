package worker

import (
	"context"
	"log"
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
func StartOutboxWorker(repo repository.LinkRepository, pub event.Publisher) {
	pgRepo, ok := repo.(*repository.PostgresLinkRepository)
	if !ok {
		log.Printf("outbox: unsupported repository type")
		return
	}

	log.Printf("outbox: worker started")
	for {
		ctx := context.Background()

		// STEP 1: claim pending outbox events using a transaction
		var events []model.OutBoxEvent
		err := pgRepo.WithTx(ctx, func(tx pgx.Tx) error {
			var err error
			events, err = pgRepo.ClaimPendingOutboxTx(ctx, tx, 10)
			return err
		})

		// if there is a failure, log it and sleep before retrying
		if err != nil {
			log.Printf("outbox: claim failed: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// Step 2: publish events (no transaction needed)
		for _, e := range events {
			log.Printf("outbox: publishing event: %v", e.ID)

			err := pub.Publish(ctx, event.Event{
				ID:   e.ID.String(),
				Type: e.EventType,
				Data: e.Payload,
			})
			if err != nil {
				log.Printf("outbox: publish failed id=%s err=%v", e.ID, err)
				continue
			}

			// Step 3: mark event as processed
			err = pgRepo.WithTx(ctx, func(tx pgx.Tx) error {
				return pgRepo.MarkOutboxProcessed(ctx, e.ID)
			})
			if err != nil {
				log.Printf("outbox: mark processed failed id=%s err=%v", e.ID, err)
				continue
			}

			log.Printf("outbox: event processed: %s", e.ID)
		}

		time.Sleep(3 * time.Second)
	}
}
