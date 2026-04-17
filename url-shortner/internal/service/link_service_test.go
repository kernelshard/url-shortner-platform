package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kernelshard/url-shortner-platform/internal/cache"
	"github.com/kernelshard/url-shortner-platform/internal/event"
	"github.com/kernelshard/url-shortner-platform/internal/model"
	"github.com/kernelshard/url-shortner-platform/internal/repository"
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

func (f *fakeRepo) InsertOutbox(ctx context.Context, event model.OutBoxEvent) error {
	return nil
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

// create no-op publisher for tests
type fakePublisher struct{}

func (f *fakePublisher) Publish(ctx context.Context, e event.Event) error {
	return nil
}

// TestCreateIdempotent tests the idempotent behavior of the Create method.
// It verifies that creating a link with the same URL returns the same short code.
func TestCreateIdempotent(t *testing.T) {
	repo := &fakeRepo{store: make(map[string]model.Link)}
	cache := cache.NewInMemoryCache()
	fakePub := &fakePublisher{}

	svc := NewLinkService(repo, cache, fakePub)
	ctx := context.Background()

	// we create two links with the same URL, and verify that they return the same short code
	link1, _ := svc.Create(ctx, "https://example.com")
	link2, _ := svc.Create(ctx, "https://example.com")

	if link1.ShortCode != link2.ShortCode {
		t.Errorf("expected same short code, got %s and %s", link1.ShortCode, link2.ShortCode)
	}
}

// TestGetByCode_UseCache tests that GetByShortCode uses the cache first before hitting the repository.
func TestGetByCode_UseCache(t *testing.T) {
	repo := &fakeRepo{
		store: make(map[string]model.Link),
	}

	cache := cache.NewInMemoryCache()
	fakePub := &fakePublisher{}
	svc := NewLinkService(repo, cache, fakePub)

	ctx := context.Background()
	link := model.Link{
		ID:          uuid.New(),
		OriginalURL: "https://example.com",
		ShortCode:   "abc123",
		CreatedAt:   time.Now(),
	}

	repo.store[link.OriginalURL] = link

	// first call -> should hit repo
	_, _ = svc.GetByShortCode(ctx, "abc123")

	// second call -> should hit cache
	_, _ = svc.GetByShortCode(ctx, "abc123")

	// as db hit only once, total call count to db shall be 1
	if repo.callCount != 1 {
		t.Errorf("expected repo to be called once, got %d", repo.callCount)
	}
}
