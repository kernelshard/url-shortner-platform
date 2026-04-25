package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/contract"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/repository"
)

type EmailService interface {
	Send(ctx context.Context, email string, subject string, body string) error
}

type emailService struct {
}

func (s *emailService) Send(ctx context.Context, email string, subject string, body string) error {
	log.Printf("sending email to %s: %s - %s", email, subject, body)
	return nil
}

func NewEmailService() *emailService {
	return &emailService{}
}

type HttpHandler struct {
	repo         repository.ProcessedEventRepository
	emailService EmailService
}

func NewHttpHandler(repo repository.ProcessedEventRepository, emailService EmailService) *HttpHandler {
	return &HttpHandler{repo: repo, emailService: emailService}
}

func (h *HttpHandler) HandleEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. decode
	var e contract.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// 2. parse UUID safely
	id, err := uuid.Parse(e.EventID)
	if err != nil {
		http.Error(w, "invalid event_id", http.StatusBadRequest)
		return
	}

	// 3. idempotency gate
	err = h.repo.Insert(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrEventAlreadyProcessed) {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// 4. process event
	switch e.Type {

	case "link.created":
		// decode payload properly
		var payload struct {
			Email string `json:"email"`
		}

		if err := json.Unmarshal(e.Data, &payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		if err := h.emailService.Send(ctx, payload.Email, "Link Created", "Your link has been created"); err != nil {
			http.Error(w, "failed to send email", http.StatusInternalServerError)
			return
		}

	default:
		// unknown event - ignore and return 200
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}
