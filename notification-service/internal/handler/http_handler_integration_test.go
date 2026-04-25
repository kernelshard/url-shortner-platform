package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/contract"
	"github.com/kernelshard/url-shortner-platform/notification-service/internal/repository"
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

// Test_Idempotency_Concurrent tests that when multiple concurrent requests with
// the same event ID are sent to the handler, only one email is sent.
func Test_Idempotency_Concurrent(t *testing.T) {
	db := repository.TestDB(t)

	repo := repository.NewPostgresProcessedEventRepository(db)

	email := &FakeEmailService{}
	handler := NewHttpHandler(repo, email)

	e := contract.Event{
		EventID: uuid.NewString(),
		Type:    "link.created",
		Data:    json.RawMessage(`{"email": "test@example.com"}`),
	}

	var wg sync.WaitGroup
	maxRequests := 50

	for range maxRequests {

		wg.Go(func() {
			req := makeRequest(t, e)
			w := httptest.NewRecorder()
			handler.HandleEvents(w, req)

			if w.Code != 200 {
				t.Errorf("expected status 200, got %d", w.Code)
			}

		})
	}

	wg.Wait()
	if email.Count() != 1 {
		t.Errorf("expected email to be sent once, but was sent %d times", email.Count())
	}
}
