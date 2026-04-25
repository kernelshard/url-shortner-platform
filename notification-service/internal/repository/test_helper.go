package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	connStr := os.Getenv("TEST_DB_URL")
	if connStr == "" {
		t.Log("TEST_DB_URL not set, using default")
		connStr = "postgres://postgres:postgres@localhost:5435/notification_test?sslmode=disable"
	}
	db, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	// ensure the connection is valid
	err = db.Ping(ctx)
	require.NoError(t, err)

	// clean the database before each test
	_, err = db.Exec(ctx, `TRUNCATE processed_events`)
	require.NoError(t, err)

	return db
}
