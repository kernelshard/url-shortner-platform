package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/repository"
)

type EventProcessor interface {
	ProcessLinkCreated(ctx context.Context, eventID uuid.UUID, email string) error
}

type eventSvc struct {
	repo repository.ProcessedEventRepository
}

func NewEventService(repo repository.ProcessedEventRepository) EventProcessor {
	return &eventSvc{repo: repo}
}

func (s *eventSvc) ProcessLinkCreated(ctx context.Context, eventID uuid.UUID, email string) error {
	return s.repo.InsertWithEmailTx(ctx, eventID, email)

}
