package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// CleanDB truncates the existing outbox_events rows
// fit for calling before each test
func CleanDB(t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	_, err := db.Exec(context.Background(), `
		TRUNCATE outbox_events, links RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
}

// SetupTestRepo sets up a test repository with a fresh database connection
func SetupTestRepo(t *testing.T) (*PostgresLinkRepository, *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	conStr := "postgres://postgres:postgres@localhost:5433/shortner_test?sslmode=disable"
	db, err := pgxpool.New(ctx, conStr)
	require.NoError(t, err)

	err = db.Ping(ctx)
	require.NoError(t, err)

	repo := &PostgresLinkRepository{db: db}
	return repo, db
}
