package http

// NOTE:
// These tests are written as narrow integration tests.
//
// We are using the real service layer, but faking external dependencies
// (like repository and cache) to check the full request flow:
//
// HTTP → handler → service → repo(fake)
//
// This helps to verify proper wiring, request parsing, and response behaviour,
// without depending on external systems like Postgres or Redis.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kernelshard/url-shortner-platform/internal/cache"
	"github.com/kernelshard/url-shortner-platform/internal/event"
	"github.com/kernelshard/url-shortner-platform/internal/model"
	"github.com/kernelshard/url-shortner-platform/internal/repository"
	"github.com/kernelshard/url-shortner-platform/internal/service"
)

type fakeRepo struct {
	// key is the original URL, value is the link model
	store     map[string]model.Link
	callCount int
}

func (f *fakeRepo) Insert(ctx context.Context, link model.Link) (model.Link, error) {
	if existing, ok := f.store[link.OriginalURL]; ok {
		return existing, repository.ErrLinkAlreadyExists
	}
	f.store[link.OriginalURL] = link
	return link, nil
}

func (f *fakeRepo) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

func (f *fakeRepo) GetByURL(ctx context.Context, url string) (model.Link, error) {
	if link, ok := f.store[url]; ok {
		return link, nil
	}
	return model.Link{}, repository.ErrLinkNotFound
}

func (f *fakeRepo) GetByShortCode(ctx context.Context, shortCode string) (model.Link, error) {
	f.callCount++

	for _, link := range f.store {
		if link.ShortCode == shortCode {
			return link, nil
		}
	}
	return model.Link{}, repository.ErrLinkNotFound
}

func (f *fakeRepo) Get(ctx context.Context, shortCode string) (model.Link, error) {
	link, ok := f.store[shortCode]
	if !ok {
		return model.Link{}, nil
	}
	return link, nil
}

func (f *fakeRepo) CreateWithOutbox(ctx context.Context, link model.Link, event model.OutBoxEvent) (model.Link, error) {
	return f.Insert(ctx, link)
}

func (f *fakeRepo) ClaimPendingOutboxTx(ctx context.Context, tx pgx.Tx, limit int) ([]model.OutBoxEvent, error) {
	return nil, nil
}

func (f *fakeRepo) GetUnprocessedOutbox(ctx context.Context, limit int) ([]model.OutBoxEvent, error) {
	return nil, nil
}

func (f *fakeRepo) MarkOutboxProcessed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (f *fakeRepo) MarkOutboxProcessedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return nil
}

func (f *fakeRepo) UpdateRetryState(ctx context.Context, id uuid.UUID, next time.Time) error {
	return nil
}

func (f *fakeRepo) InsertOutboxTx(ctx context.Context, tx pgx.Tx, event model.OutBoxEvent) error {
	return nil
}

func (f *fakeRepo) UpdateRetryStateTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, next time.Time) error {
	return nil
}

func (f *fakeRepo) MarkOutboxDeadTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return nil
}

// create no-op publisher for tests
type fakePublisher struct{}

func (f *fakePublisher) Publish(ctx context.Context, e event.Event) error {
	return nil
}

// TestCreateShortURL_Handler tests the CreateShortURL handler
func TestCreateShortURL_Handler(t *testing.T) {
	reqBody := `{ "url": "https://example.com" }`
	req := httptest.NewRequest(http.MethodPost, "/urls", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// use real service + fake repo, check the top level file doc/comment
	repo := &fakeRepo{store: make(map[string]model.Link)}
	inMemCache := cache.NewInMemoryCache()
	fakePub := &fakePublisher{}
	svc := service.NewLinkService(repo, inMemCache, fakePub)
	h := NewHandler(svc)

	h.CreateShortURL(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %v", w.Code)
	}

	var resp map[string]any
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	shortURLRaw, ok := resp["short_url"]
	if !ok {
		t.Fatalf("expected short_url to be present in response, got %v", resp)
	}

	shortURL, ok := shortURLRaw.(string)
	if !ok || shortURL == "" {
		t.Fatalf("expected short_url to be a non-empty string, got %v", shortURLRaw)
	}

	if !strings.Contains(shortURL, "localhost:8080/") {
		t.Errorf("unexpected short_url format, got %v", shortURL)
	}

}

func TestCreateShortURL_Idempotent(t *testing.T) {
	reqBody := `{"url": "https://example.com"}`

	repo := &fakeRepo{store: make(map[string]model.Link)}
	inMemCache := cache.NewInMemoryCache()
	fakePub := &fakePublisher{}
	svc := service.NewLinkService(repo, inMemCache, fakePub)
	h := NewHandler(svc)

	// first request
	req1 := httptest.NewRequest(http.MethodPost, "/urls", strings.NewReader(reqBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	h.CreateShortURL(w1, req1)

	// second req
	req2 := httptest.NewRequest(http.MethodPost, "/urls", strings.NewReader(reqBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.CreateShortURL(w2, req2)

	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Errorf("expected status 200 for both requests, got %v and %v", w1.Code, w2.Code)
	}

	var resp1, resp2 map[string]any
	err := json.NewDecoder(w1.Body).Decode(&resp1)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	err = json.NewDecoder(w2.Body).Decode(&resp2)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	shortURL1Raw, ok := resp1["short_url"]
	if !ok {
		t.Fatalf("expected short_url to be present in response, got %v", resp1)
	}
	shortURL2Raw, ok := resp2["short_url"]
	if !ok {
		t.Fatalf("expected short_url to be present in response, got %v", resp2)
	}

	shortURL1, ok := shortURL1Raw.(string)
	if !ok || shortURL1 == "" {
		t.Fatalf("expected short_url to be a non-empty string, got %v", shortURL1Raw)
	}
	shortURL2, ok := shortURL2Raw.(string)
	if !ok || shortURL2 == "" {
		t.Fatalf("expected short_url to be a non-empty string, got %v", shortURL2Raw)
	}

	if shortURL1 != shortURL2 {
		t.Errorf("expected short_url to be the same for both requests, got %v and %v", shortURL1, shortURL2)
	}
}

func TestCreateShortURL_InvalidInput(t *testing.T) {
	reqBody := `{"url": ""}`
	req := httptest.NewRequest(http.MethodPost, "/urls", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	repo := &fakeRepo{store: make(map[string]model.Link)}
	inMemCache := cache.NewInMemoryCache()
	fakePub := &fakePublisher{}
	svc := service.NewLinkService(repo, inMemCache, fakePub)
	h := NewHandler(svc)

	h.CreateShortURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid input, got %v", w.Code)
	}
}
