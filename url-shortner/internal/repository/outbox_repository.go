package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kernelshard/url-shortner-platform/internal/model"
)

// OutboxRepository defines the interface for the outbox repository.
type OutboxRepository interface {
	WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error
	// ClaimPendingOutboxTx claims a batch of pending outbox events for processing
	ClaimPendingOutboxTx(ctx context.Context, tx pgx.Tx, limit int) ([]model.OutBoxEvent, error)
	// MarkOutboxProcessedTx marks a batch of outbox events as processed.
	MarkOutboxProcessedTx(ctx context.Context, tx pgx.Tx, eventIDs uuid.UUID) error
	// UpdateRetryState updates the retry state of an outbox event.
	UpdateRetryState(ctx context.Context, id uuid.UUID, next time.Time) error
}

// PostgresLinkRepository is a concrete implementation of LinkRepository using PostgreSQL.
type PostgresLinkRepository struct {
	db *pgxpool.Pool
}

func NewPostgresLinkRepository(db *pgxpool.Pool) *PostgresLinkRepository {
	return &PostgresLinkRepository{db: db}
}

// DB returns the underlying pgxpool.Pool.
func (r *PostgresLinkRepository) DB() *pgxpool.Pool {
	return r.db
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

// ClaimPendingOutboxTx atomically claims unprocessed outbox eventsl, each event is processed by only one worker.
//
// 1. selects unprocessed and unclaimes events
// 2. skips rows locked by other transactions
// 3. mark them as claimed by updating the `claimed_at` column
func (r *PostgresLinkRepository) ClaimPendingOutboxTx(ctx context.Context, tx pgx.Tx, limit int) ([]model.OutBoxEvent, error) {
	// skip already claimed events and rows locked by other transactions
	query := `
		SELECT id, event_type, payload, created_at, processed, next_retry_at, retry_count, claimed_at
		FROM outbox_events
		WHERE processed = false
		AND (
		    claimed_at IS NULL
			OR claimed_at < NOW() - INTERVAL '1 minute'
		)
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
		if err := rows.Scan(
			&e.ID, &e.EventType,
			&e.Payload, &e.CreatedAt,
			&e.Processed, &e.NextRetryAt,
			&e.RetryCount, &e.ClaimedAt); err != nil {
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
			SET claimed_at = NOW()
			WHERE id = ANY($1)
			`, ids)
		if err != nil {
			return nil, err
		}
	}

	return events, nil
}

// InsertOutbox inserts an outbox event into the database.
func (r *PostgresLinkRepository) InsertOutbox(ctx context.Context, event model.OutBoxEvent) error {
	query := `
		INSERT INTO outbox_events (id, event_type, payload, created_at, processed, next_retry_at, retry_count, claimed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		event.ID,
		event.EventType,
		event.Payload,
		event.CreatedAt,
		event.Processed,
		event.NextRetryAt,
		event.RetryCount,
		event.ClaimedAt,
	)
	return err
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

// GetUnprocessedOutbox retrieves unprocessed outbox events from the database.
func (r *PostgresLinkRepository) GetUnprocessedOutbox(ctx context.Context, limit int) ([]model.OutBoxEvent, error) {
	query := `
		SELECT id, event_type, payload, created_at, processed, next_retry_at, retry_count, claimed_at
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
		if err := rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.CreatedAt, &e.Processed, &e.NextRetryAt, &e.RetryCount, &e.ClaimedAt); err != nil {
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
func (r *PostgresLinkRepository) UpdateRetryState(ctx context.Context, id uuid.UUID, next time.Time) error {
	query := `UPDATE outbox_events
			  SET retry_count = retry_count + 1,
					next_retry_at = $1
			  WHERE id = $2`
	_, err := r.db.Exec(ctx, query, next, id)
	return err
}
