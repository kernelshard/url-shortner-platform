package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/contract"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/repository"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/service"
)

type HttpHandler struct {
	eventSvc service.EventProcessor
}

func NewHttpHandler(eventSvc service.EventProcessor) *HttpHandler {
	return &HttpHandler{eventSvc: eventSvc}
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

		// insert pending email delivery to be processed by a worker later
		err = h.eventSvc.ProcessLinkCreated(ctx, id, payload.Email)
		if err != nil {
			//
			if errors.Is(err, repository.ErrEventAlreadyProcessed) {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

	default:
		// unknown event - ignore and return 200
		log.Printf("received unknown event type: %s returning 200", e.Type)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}
