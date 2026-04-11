package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kernelshard/url-shortner-platform/internal/model"
)

var (
	ErrLinkAlreadyExists = errors.New("link already exists")
	ErrShortCodeConflict = errors.New("short code conflict")
	ErrLinkNotFound      = errors.New("link not found")
)

// LinkRepository defines the interface for managing Link records in the data store.
type LinkRepository interface {
	// Insert inserts a new Link record into the database.
	// It returns an error if a link with the same original URL already exists
	// or if a short code conflict occurs.
	Insert(ctx context.Context, link model.Link) (model.Link, error)
	GetByURL(ctx context.Context, originalURL string) (model.Link, error)
	GetByShortCode(ctx context.Context, shortCode string) (model.Link, error)

	// OutboxEvents
	InsertOutbox(ctx context.Context, event model.OutBoxEvent) error
	GetUnprocessedOutbox(ctx context.Context, limit int) ([]model.OutBoxEvent, error)
	MarkOutboxProcessed(ctx context.Context, id uuid.UUID) error
}

// PostgresLinkRepository is a concrete implementation of LinkRepository using PostgreSQL.
type PostgresLinkRepository struct {
	db *pgxpool.Pool
}

func NewPostgresLinkRepository(db *pgxpool.Pool) *PostgresLinkRepository {
	return &PostgresLinkRepository{db: db}
}

// Insert inserts a new Link record into the database.
// It returns an error if a link with the same original URL already exists
// or if a short code conflict occurs.
func (r *PostgresLinkRepository) Insert(ctx context.Context, link model.Link) (model.Link, error) {
	query := `
		INSERT INTO links (id, original_url, short_code, created_at, expires_at)
	VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id uuid.UUID
	err := r.db.QueryRow(
		ctx, query,
		link.ID,
		link.OriginalURL,
		link.ShortCode,
		link.CreatedAt,
		link.ExpiresAt,
	).Scan(&id)

	if err != nil {
		// Handle unique constraint violations for original_url and short_code
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.ConstraintName {
			case "links_original_url_key":
				return model.Link{}, ErrLinkAlreadyExists
			case "links_short_code_key":
				return model.Link{}, ErrShortCodeConflict
			}
		}
		return model.Link{}, err
	}

	link.ID = id
	return link, nil
}

// GetByURL retrieves a Link record by its original URL. It returns an error if no record is found.
func (r *PostgresLinkRepository) GetByURL(ctx context.Context, originalURL string) (model.Link, error) {
	query := `
		SELECT id, original_url, short_code, created_at, expires_at
		FROM links
		WHERE original_url = $1
	`

	var link model.Link
	err := r.db.QueryRow(ctx, query, originalURL).Scan(
		&link.ID,
		&link.OriginalURL,
		&link.ShortCode,
		&link.CreatedAt,
		&link.ExpiresAt,
	)
	if err != nil {
		return model.Link{}, err
	}
	return link, nil
}

// GetByShortCode retrieves a Link record by its short code. It returns an error if no record is found.
func (r *PostgresLinkRepository) GetByShortCode(ctx context.Context, shortCode string) (model.Link, error) {
	query := `
		SELECT id, original_url, short_code, created_at, expires_at
		FROM links
		WHERE short_code = $1
	`

	var link model.Link
	err := r.db.QueryRow(ctx, query, shortCode).Scan(
		&link.ID,
		&link.OriginalURL,
		&link.ShortCode,
		&link.CreatedAt,
		&link.ExpiresAt,
	)
	// If no rows are returned, we consider it a "not found" error
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.Link{}, ErrLinkNotFound
		}
		return model.Link{}, err
	}
	return link, nil
}

// InsertOutbox inserts an outbox event into the database.
func (r *PostgresLinkRepository) InsertOutbox(ctx context.Context, event model.OutBoxEvent) error {
	query := `
		INSERT INTO outbox_events (id, event_type, payload, created_at, processed, next_retry_at, retry_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		event.ID,
		event.EventType,
		event.Payload,
		event.CreatedAt,
		event.Processed,
		event.NextRetryAt,
		event.RetryCount,
	)
	return err
}

// GetUnprocessedOutbox retrieves unprocessed outbox events from the database.
func (r *PostgresLinkRepository) GetUnprocessedOutbox(ctx context.Context, limit int) ([]model.OutBoxEvent, error) {
	query := `
		SELECT id, event_type, payload, created_at, processed, next_retry_at, retry_count
		FROM outbox_events
		WHERE processed = false
		AND next_retry_at <= NOW()
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.OutBoxEvent

	for rows.Next() {
		var e model.OutBoxEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.CreatedAt, &e.Processed, &e.NextRetryAt, &e.RetryCount); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil

}

// MarkOutboxProcessed marks an outbox event as processed in the database.
func (r *PostgresLinkRepository) MarkOutboxProcessed(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE outbox_events
		SET processed = true
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// MarkOutboxProcessedTx marks an outbox event as processed in the database within a transaction.
func (r *PostgresLinkRepository) MarkOutboxProcessedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	query := `
		UPDATE outbox_events
		SET processed = true
		WHERE id = $1
	`
	_, err := tx.Exec(ctx, query, id)
	return err
}

// UpdateNextRetry updates the next retry time for an outbox event in the database.
func (r *PostgresLinkRepository) UpdateNextRetry(ctx context.Context, id uuid.UUID, next time.Time) error {
	query := `UPDATE outbox_events SET next_retry_at = $1 WHERE id=$2`
	_, err := r.db.Exec(ctx, query, next, id)
	return err
}

// WithTx executes a function within a transaction.
func (r *PostgresLinkRepository) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// InsertTx inserts a link into the database within a transaction.
func (r *PostgresLinkRepository) InsertTx(ctx context.Context, tx pgx.Tx, link model.Link) (model.Link, error) {
	query := `
		INSERT INTO links (id, original_url, short_code, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id uuid.UUID
	err := tx.QueryRow(ctx, query,
		link.ID,
		link.OriginalURL,
		link.ShortCode,
		link.CreatedAt,
		link.ExpiresAt,
	).Scan(&id)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.ConstraintName {
			case "links_original_url_key":
				return model.Link{}, ErrLinkAlreadyExists
			case "links_short_code_key":
				return model.Link{}, ErrShortCodeConflict
			}
		}
		return model.Link{}, err
	}
	link.ID = id
	return link, nil
}

// InsertOutboxTx inserts an outbox event into the database within a transaction.
func (r *PostgresLinkRepository) InsertOutboxTx(ctx context.Context, tx pgx.Tx, event model.OutBoxEvent) error {
	query := `
		INSERT INTO outbox_events (id, event_type, payload, created_at, processed, next_retry_at, retry_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := tx.Exec(ctx, query,
		event.ID,
		event.EventType,
		event.Payload,
		event.CreatedAt,
		event.Processed,
		event.NextRetryAt,
		event.RetryCount,
	)
	return err
}

// ClaimPendingOutboxTx atomically claims unprocessed outbox eventsl, each event is processed by only one worker.
//
// 1. selects unprocessed and unclaimes events
// 2. skips rows locked by other transactions
// 3. mark them as claimed by updating the `claimed_at` column
func (r *PostgresLinkRepository) ClaimPendingOutboxTx(ctx context.Context, tx pgx.Tx, limit int) ([]model.OutBoxEvent, error) {
	// skip already claimed events and rows locked by other transactions
	query := `
		SELECT id, event_type, payload, created_at, processed, next_retry_at, retry_count
		FROM outbox_events
		WHERE processed = false
		AND claimed_at IS NULL
		AND next_retry_at <= NOW()
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`
	rows, err := tx.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.OutBoxEvent
	for rows.Next() {
		var e model.OutBoxEvent
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.CreatedAt, &e.Processed, &e.NextRetryAt, &e.RetryCount); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	// returned value length in rows peropery
	ids := make([]uuid.UUID, 0, len(events))
	for _, e := range events {
		ids = append(ids, e.ID)
	}

	// Mark events as claimed as it has been retrieved for processing
	if len(ids) > 0 {
		_, err := tx.Exec(ctx, `
			UPDATE outbox_events
			SET claimed_at IS NULL OR claimed_at < NOW() - INTERVAL '1 minute'
			WHERE id = ANY($1)
			`, ids)
		if err != nil {
			return nil, err
		}
	}

	return events, nil
}
