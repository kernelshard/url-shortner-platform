package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	CreateWithOutbox(ctx context.Context, link model.Link, event model.OutBoxEvent) (model.Link, error)
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

// CreateWithOutbox inserts a link into the database with an outbox event.
func (r *PostgresLinkRepository) CreateWithOutbox(ctx context.Context, link model.Link, event model.OutBoxEvent) (model.Link, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return model.Link{}, err
	}
	defer tx.Rollback(ctx)

	// Insert the link into the database within the transaction
	link, err = r.insertTx(ctx, tx, link)
	if err != nil {
		return model.Link{}, err
	}

	// Insert the outbox event into the database within the transaction
	err = r.insertOutboxTx(ctx, tx, event)
	if err != nil {
		return model.Link{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Link{}, err
	}

	return link, nil
}

// insertTx inserts a link into the database within a transaction.
func (r *PostgresLinkRepository) insertTx(ctx context.Context, tx pgx.Tx, link model.Link) (model.Link, error) {
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
