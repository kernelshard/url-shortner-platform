package worker

import (
	"context"
	"time"

	"github.com/kernelshard/url-shortner-platform/notification-service/internal/handler"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/repository"
)

func RunWorker(ctx context.Context, repo repository.EmailDeliveryRepository, emailService handler.EmailService) {
	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processPendingBatchEmails(ctx, repo, emailService)
		}
	}
}

// processPendingBatchEmails fetches a batch of pending email deliveries and attempts to send them.
func processPendingBatchEmails(ctx context.Context, repo repository.EmailDeliveryRepository, emeilSvc handler.EmailService) {
	items, err := repo.FetchPending(ctx, 10)
	if err != nil {
		return
	}

	for _, item := range items {
		err := emeilSvc.Send(ctx, item.Email, "Link Created", "Your link has been created successfully.")
		if err != nil {
			_ = repo.MarkFailed(ctx, item.EventID, err.Error())
			continue
		}
		_ = repo.MarkSent(ctx, item.EventID)
	}
}
