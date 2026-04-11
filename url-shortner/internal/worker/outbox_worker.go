package worker

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/kernelshard/url-shortner-platform/internal/event"
	"github.com/kernelshard/url-shortner-platform/internal/repository"
)

func StartOutboxWorker(repo repository.LinkRepository, pub event.Publisher) {
	pgRepo, ok := repo.(*repository.PostgresLinkRepository)
	if !ok {
		log.Printf("outbox: unsupported repository type")
		return
	}

	log.Printf("outbox: worker started")
	for {
		ctx := context.Background()

		err := pgRepo.WithTx(ctx, func(tx pgx.Tx) error {
			// lock rows for update, skipping locked rows for concurrency
			// scenario: multiple workers may run concurrently, but only one should process each event
			events, err := pgRepo.ClaimPendingOutboxTx(ctx, tx, 10)
			if err != nil {
				return err
			}

			log.Printf("outbox: fetched count=%d", len(events))

			for _, e := range events {
				log.Printf("outbox: publishing type=%s id=%s", e.EventType, e.ID)

				err := pub.Publish(ctx, event.Event{
					Type: e.EventType,
					Data: e.Payload,
				})

				if err != nil {
					log.Printf("outbox: publish failed id=%s err=%v", e.ID, err)
					continue
				}

				log.Printf("outbox: publish success id=%s", e.ID)

				err = pgRepo.MarkOutboxProcessedTx(ctx, tx, e.ID)
				if err != nil {
					log.Printf("outbox: mark processed failed id=%s err=%v", e.ID, err)
					return err
				}

				log.Printf("outbox: marked processed id=%s", e.ID)
			}

			return nil
		})
		if err != nil {
			log.Printf("outbox: tx error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		time.Sleep(time.Second * 3)
	}
}
