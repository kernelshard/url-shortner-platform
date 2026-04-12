package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewHTTPHandler(db *pgxpool.Pool) *HTTPHandler {
	return &HTTPHandler{db: db}
}

type HTTPHandler struct {
	db *pgxpool.Pool
}

func (h *HTTPHandler) HandleEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// parse request body
	var req struct {
		EventID   string          `json:"event_id"`
		EventType string          `json:"event_type"`
		Data      json.RawMessage `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	// 1. idempotency check (INSERT-first)
	_, err := h.db.Exec(ctx, `
		INSERT INTO processed_events (event_id)
		VALUES ($1)
	`, req.EventID)

	if err != nil {
		if isUniqueViolation(err) {
			log.Printf("duplicate event ignored id %s", req.EventID)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// 2. process event (for now, just log)
	log.Printf("processed event: id=%s type=%s", req.EventID, req.EventType)
	w.WriteHeader(http.StatusOK)
}

// isUniqueViolation checks if the error is a unique violation error
func isUniqueViolation(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == "23505"
	}
	return false
}
