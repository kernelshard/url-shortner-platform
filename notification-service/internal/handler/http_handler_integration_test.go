package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/contract"
)

// FakeEmailService is a thread-safe fake implementation of the EmailService interface
// that counts the number of times the Send method is called.
type FakeEmailService struct {
	mu    sync.Mutex
	count int
}

func (f *FakeEmailService) Send(ctx context.Context, email string, subject string, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.count++
	return nil
}

func (f *FakeEmailService) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.count
}

type mockEventService struct {
	err    error
	called bool
}

func (m *mockEventService) ProcessLinkCreated(ctx context.Context, eventID uuid.UUID, email string) error {
	m.called = true
	return m.err
}

// Test_Idempotency_Concurrent tests that when multiple concurrent requests with
// the same event ID are sent to the handler, only one email is sent.
func Test_Idempotency_Concurrent(t *testing.T) {
	svc := &mockEventService{}
	handler := NewHttpHandler(svc)

	e := contract.Event{
		EventID: uuid.NewString(),
		Type:    "link.created",
		Data:    json.RawMessage(`{"email": "test@example.com"}`),
	}

	var wg sync.WaitGroup
	maxRequests := 50

	for range maxRequests {
		wg.Add(1)

		go func() {
			defer wg.Done()

			req := makeRequest(t, e)
			w := httptest.NewRecorder()

			handler.HandleEvents(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
		}()
	}
	wg.Wait()

}
