//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// connectPostgres retries for up to 10 attempts so `docker compose up`
// doesn't require a manual restart while Postgres is still initializing.
// That retry loop is the actual behavior worth testing here — a real
// Postgres container stands in for the database instead of mocking the
// pgx driver, matching how internal/repository's integration tests work.
func TestConnectPostgres_ConnectsToRealPostgres(t *testing.T) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("orders"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := connectPostgres(ctx, connStr, discardLogger())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx))
}

func TestConnectPostgres_GivesUpAfterRetriesOnBadDSN(t *testing.T) {
	if testing.Short() {
		t.Skip("exercises the full 10-attempt retry loop (~20s); skipped with -short")
	}

	// A syntactically valid but unreachable DSN makes pgxpool.New succeed
	// immediately while every Ping fails, driving the function through its
	// full retry loop (10 attempts, 2s apart) before it gives up and
	// returns the last error.
	_, err := connectPostgres(context.Background(), "postgres://postgres:postgres@localhost:1/nope?sslmode=disable&connect_timeout=1", discardLogger())

	require.Error(t, err)
}
