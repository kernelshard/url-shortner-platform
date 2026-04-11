package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	for _ = range 3 {
		shortCode := generateShortCode()

		id := uuid.New()
		link := model.Link{
			ID:          id,
			OriginalURL: originalURL,
			ShortCode:   shortCode,
			CreatedAt:   time.Now(),
		}

		var (
			created model.Link
			err     error
		)

		// transactional path
		if pgRepo, ok := s.repo.(*repository.PostgresLinkRepository); ok {
			log.Printf("create: tx path url=%s", originalURL)

			err = pgRepo.WithTx(ctx, func(tx pgx.Tx) error {
				var err error

				created, err = pgRepo.InsertTx(ctx, tx, link)
				if err != nil {
					return err
				}

				payload, err := json.Marshal(created)
				if err != nil {
					return err
				}

				return pgRepo.InsertOutboxTx(ctx, tx, model.OutBoxEvent{
					ID:        uuid.New(),
					EventType: "link.created",
					Payload:   payload,
				})
			})
		} else {
			// fallback path (tests)
			log.Printf("create: fallback path url=%s", originalURL)

			created, err = s.repo.Insert(ctx, link)
			if err == nil {
				payload, _ := json.Marshal(created)
				_ = s.repo.InsertOutbox(ctx, model.OutBoxEvent{
					ID:        uuid.New(),
					EventType: "link.created",
					Payload:   payload,
				})
			}
		}

		// idempotent case
		if errors.Is(err, repository.ErrLinkAlreadyExists) {
			log.Printf("create: idempotent url=%s", originalURL)
			return s.repo.GetByURL(ctx, originalURL)
		}

		// retry case
		if errors.Is(err, repository.ErrShortCodeConflict) {
			log.Printf("create: collision retry url=%s", originalURL)
			lastErr = err
			continue
		}

		// failure
		if err != nil {
			log.Printf("create: failed url=%s err=%v", originalURL, err)
			return model.Link{}, err
		}

		// success
		log.Printf("create: success id=%s short=%s", created.ID, created.ShortCode)
		return created, nil
	}

	log.Printf("create: exhausted retries url=%s err=%v", originalURL, lastErr)
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

// generateShortCode creates a short, URL-safe identifier using high-entropy randomness.
//
// Design decision:
// - Uses crypto/rand to ensure unpredictability of generated codes.
// - Prevents enumeration attacks where attackers guess valid short URLs.
// - Collisions are handled at the DB layer with retry logic, so generator prioritizes entropy over determinism.
//
// Invariant:
// - Generated codes must be hard to predict under concurrent access.
func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			log.Printf("failed to generate random number: %v", err)
			panic(err)
		}
		b[i] = charset[n.Int64()]
	}

	return string(b)
}
