package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailDelivery struct {
	EventID uuid.UUID `db:"event_id"`
	Email   string    `db:"email"`
}

// EmailDeliveryRepository defines the interface for managing email delivery records.
type EmailDeliveryRepository interface {
	InsertPending(ctx context.Context, eventID uuid.UUID, email string) error
	FetchPending(ctx context.Context, limit int) ([]EmailDelivery, error)
	MarkSent(ctx context.Context, eventID uuid.UUID) error
	MarkFailed(ctx context.Context, eventID uuid.UUID, errMsg string) error
}

type pgEmailDeliveryRepo struct {
	db *pgxpool.Pool
}

// NewPostgresEmailDeliveryRepository creates a new instance of postgresEmailDeliveryRepository with the given database connection pool.
func NewPostgresEmailDeliveryRepository(db *pgxpool.Pool) EmailDeliveryRepository {
	return &pgEmailDeliveryRepo{db: db}
}

// InsertPending inserts a pending email delivery record for the given event ID and email address.
func (r *pgEmailDeliveryRepo) InsertPending(ctx context.Context, eventID uuid.UUID, email string) error {
	query := `
	    INSERT INTO email_deliveries (event_id, email)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING
		`
	_, err := r.db.Exec(ctx, query, eventID, email)
	return err
}

// FetchPending retrieves a list of pending email deliveries up to the specified limit, locking the selected
// rows for update to prevent concurrent processing.
func (r *pgEmailDeliveryRepo) FetchPending(ctx context.Context, limit int) ([]EmailDelivery, error) {
	query := `
	    SELECT event_id, email
		FROM email_deliveries
		WHERE status IN ('pending', 'failed')
		    AND retry_count < 5
			AND (next_retry_at IS NULL OR next_retry_at < NOW())
		FOR UPDATE SKIP LOCKED
		LIMIT $1
		`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveriesResult []EmailDelivery
	for rows.Next() {
		var e EmailDelivery
		if err := rows.Scan(&e.EventID, &e.Email); err != nil {
			return nil, err
		}
		deliveriesResult = append(deliveriesResult, e)
	}
	return deliveriesResult, nil

}

// MarkSent updates the status of the email delivery record associated with the given event ID to "sent" and sets the sent timestamp.
func (r *pgEmailDeliveryRepo) MarkSent(ctx context.Context, eventID uuid.UUID) error {
	query := `
	    UPDATE email_deliveries
		SET status = 'sent',
		    sent_at = NOW(),
		WHERE event_id = $1
		`
	_, err := r.db.Exec(ctx, query, eventID)
	return err
}

// MarkFailed updates the status of the email delivery record associated with the given event ID to "failed",
// increments the retry count, and records the error message.
func (r *pgEmailDeliveryRepo) MarkFailed(ctx context.Context, eventID uuid.UUID, errMsg string) error {
	query := `
	    UPDATE email_deliveries
		SET status = 'failed',
		    retry_count = retry_count + 1,
			last_error = $2,
		where event_id = $1
		`
	_, err := r.db.Exec(ctx, query, eventID, errMsg)
	return err
}
