package worker

import (
	"context"
	"log"
	"time"

	"github.com/kernelshard/url-shortner-platform/internal/event"
	"github.com/kernelshard/url-shortner-platform/internal/repository"
)

func StartOutboxWorker(repo repository.LinkRepository, pub event.Publisher) {
	log.Printf("outbox: worker started")
	for {
		ctx := context.Background()

		events, err := repo.GetUnprocessedOutbox(ctx, 10)
		if err != nil {
			log.Printf("outbox: fetch error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		log.Printf("outbox: fetched count=%d", len(events))

		for _, e := range events {
			log.Printf("outbox: publishing type=%s id=%s", e.Type, e.ID)

			err := pub.Publish(ctx, event.Event{
				Type: e.Type,
				Data: e.Payload,
			})

			if err != nil {
				log.Printf("outbox: publish failed id=%s err=%v", e.ID, err)
				continue // retry later
			}

			log.Printf("outbox: publish success id=%s", e.ID)

			err = repo.MarkOutboxProcessed(ctx, e.ID)
			if err != nil {
				log.Printf("outbox: mark processed failed id=%s err=%v", e.ID, err)
			} else {
				log.Printf("outbox: marked processed id=%s", e.ID)
			}
		}

		time.Sleep(time.Second * 3)
	}
}
