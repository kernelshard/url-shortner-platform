package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/contract"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	insertErr error
}

func (m *mockRepo) Insert(ctx context.Context, id uuid.UUID) error {
	return m.insertErr
}

func (m *mockRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockRepo) InsertWithEmailTx(ctx context.Context, id uuid.UUID, email string) error {
	return m.insertErr
}

type mockEmailService struct {
	called bool
	err    error
}

func (m *mockEmailService) Send(ctx context.Context, email string, subject string, body string) error {
	m.called = true
	return m.err
}

func makeRequest(t *testing.T, e contract.Event) *http.Request {
	t.Helper()

	body, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	return req
}

// TestHandleEvents_InvalidBody tests that the handler returns a 400 Bad Request when the request body is not valid JSON.
func TestHandleEvents_InvalidBody(t *testing.T) {
	eventSvc := &mockEventService{}
	handler := NewHttpHandler(eventSvc)

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler.HandleEvents(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleEvents_InvalidEventID tests that the handler returns a 400 Bad Request when the event_id is not a valid UUID.
func TestHandleEvents_InvalidEventID(t *testing.T) {
	handler := NewHttpHandler(&mockEventService{})

	e := contract.Event{
		EventID: "invalid-uuid",
		Type:    "link.created",
	}
	req := makeRequest(t, e)
	w := httptest.NewRecorder()

	handler.HandleEvents(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, "expected 400 for invalid event_id")
}

func TestHandleEvents_EventAlreadyProcessed(t *testing.T) {
	eventService := &mockEventService{
		err: repository.ErrEventAlreadyProcessed,
	}
	handler := NewHttpHandler(eventService)

	e := contract.Event{
		EventID: uuid.New().String(),
		Type:    "link.created",
		Data:    json.RawMessage(`{"email": "test@example.com"}`),
	}
	req := makeRequest(t, e)
	w := httptest.NewRecorder()

	handler.HandleEvents(w, req)
	require.Equal(t, http.StatusOK, w.Code, "expected 200 OK for already processed event")
	require.True(t, eventService.called, "event service should not be called for already processed event")
}

func TestHandleEvents_EmailServiceError(t *testing.T) {
	eventSvc := &mockEventService{err: assert.AnError}
	handler := NewHttpHandler(eventSvc)

	e := contract.Event{
		EventID: uuid.New().String(),
		Type:    "link.created",
		Data:    json.RawMessage(`{"email": "test@example.com"}`),
	}

	req := makeRequest(t, e)
	w := httptest.NewRecorder()

	handler.HandleEvents(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code, "expected 500 Internal Server Error when email service fails")
}

func TestHandleEvents_Success(t *testing.T) {
	eventSvc := &mockEventService{}
	handler := NewHttpHandler(eventSvc)

	e := contract.Event{
		EventID: uuid.New().String(),
		Type:    "link.created",
		Data:    json.RawMessage(`{"email": "test@example.com"}`),
	}

	req := makeRequest(t, e)
	w := httptest.NewRecorder()

	handler.HandleEvents(w, req)

	require.Equal(t, http.StatusOK, w.Code, "expected 200 OK for successful event processing")
	require.True(t, eventSvc.called, "email service should be called for successful event")
}
