package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/kernelshard/url-shortner-platform/internal/cache"
	"github.com/kernelshard/url-shortner-platform/internal/event"
	"github.com/kernelshard/url-shortner-platform/internal/model"
	"github.com/kernelshard/url-shortner-platform/internal/repository"
	"golang.org/x/sync/singleflight"
)

// LinkService provides business logic for managing shortened URLs.
type LinkService interface {
	Create(ctx context.Context, originalURL string) (model.Link, error)
	GetByShortCode(ctx context.Context, shortCode string) (model.Link, error)
}

// linkService implements the LinkService interface.
type linkService struct {
	repo  repository.LinkRepository
	cache cache.Cache
	sf    singleflight.Group // used for deduplicating concurrent requests
	pub   event.Publisher
}

// NewLinkService creates a new link service with the given link repository and cache.
func NewLinkService(repo repository.LinkRepository, cache cache.Cache, pub event.Publisher) LinkService {
	return &linkService{repo: repo, cache: cache, pub: pub}
}

// Create generates a short code for the given original URL and stores it in the repository.
// It handles potential conflicts by retrying with a new short code up to 3 times.
//   - If the same original URL already exists, it returns the existing record (idempotent).
//   - If a short code collision occurs, it retries with a new code.
//   - For other errors, it returns the error immediately.
func (s *linkService) Create(ctx context.Context, originalURL string) (model.Link, error) {
	var lastErr error

	for range 3 {
		shortCode := generateShortCode()

		id := uuid.New()
		link := model.Link{
			ID:          id,
			OriginalURL: originalURL,
			ShortCode:   shortCode,
			CreatedAt:   time.Now(),
		}

		created, err := s.repo.Create(ctx, link)
		if err == nil {
			if s.pub != nil {
				err = s.pub.Publish(ctx, event.Event{
					Type: "link.created",
					Data: created,
				})
				if err != nil {
					log.Printf("failed to publish event link.created: %v", err)
				}
			}
			return created, nil
		}

		// Case 1: same URL already exists -> idempotent fetch
		if errors.Is(err, repository.ErrLinkAlreadyExists) {
			return s.repo.GetByURL(ctx, originalURL)
		}

		// Case 2: short_code collision -> retry with new code
		// Rtrying is only for short code conflict, other DB errors should fail immediately
		if errors.Is(err, repository.ErrShortCodeConflict) {
			lastErr = err
			continue
		}
		// Other DB error -> fail
		return model.Link{}, err
	}

	// exhausted retries
	return model.Link{}, lastErr
}

// GetByShortCode retrieves a link by its short code.
// If the link is not found, it returns repository.ErrLinkNotFound.
func (s *linkService) GetByShortCode(ctx context.Context, shortCode string) (model.Link, error) {
	// 1. Check cache
	val, ok := s.cache.Get(ctx, shortCode)
	if ok {
		log.Printf("cache hit for short code: %s", shortCode)
		return model.Link{
			ShortCode:   shortCode,
			OriginalURL: val,
		}, nil
	}

	log.Printf("cache miss for short code: %s", shortCode)

	// 2. Collapse concurrent requests for the same short code
	v, err, _ := s.sf.Do(shortCode, func() (any, error) {
		// double-check cache
		if val, ok := s.cache.Get(ctx, shortCode); ok {
			return model.Link{
				ShortCode:   shortCode,
				OriginalURL: val,
			}, nil
		}

		link, err := s.repo.GetByShortCode(ctx, shortCode)
		if err != nil {
			return model.Link{}, err
		}

		// 3. Store in cache
		s.cache.Set(ctx, shortCode, link.OriginalURL)
		return link, nil
	})

	// Case 1: link not found -> return ErrLinkNotFound
	// Case 2: other DB error -> fail
	if err != nil {
		if err == repository.ErrLinkNotFound {
			return model.Link{}, repository.ErrLinkNotFound
		}
		return model.Link{}, err
	}

	return v.(model.Link), nil
}

// generateShortCode generates a new short code for a link.
func generateShortCode() string {
	// TODO: replace with real logic (random/base62 etc.)
	return "abc123"
}
