package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEventAlreadyProcessed = errors.New("event already processed")
)

type ProcessedEventRepository interface {
	Insert(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type postgresProcessedEventRepository struct {
	db *pgxpool.Pool
}

func NewPostgresProcessedEventRepository(db *pgxpool.Pool) ProcessedEventRepository {
	return &postgresProcessedEventRepository{db: db}
}

// Insert tries to mark event as processed.
// If already processed, returns ErrEventAlreadyProcessed.
func (r *postgresProcessedEventRepository) Insert(ctx context.Context, id uuid.UUID) error {
	query := `
		INSERT INTO processed_events (event_id)
		VALUES ($1)
		ON CONFLICT (event_id) DO NOTHING
	`

	res, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return ErrEventAlreadyProcessed
	}

	return nil
}

// Exists checks if event is already processed.
func (r *postgresProcessedEventRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM processed_events WHERE event_id = $1
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
