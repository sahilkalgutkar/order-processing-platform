//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/domain"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/repository"
)

// setupPostgres spins up a real Postgres in a container and applies the same
// schema as migrations/0001_create_orders.up.sql, so OrderRepository is
// exercised against the real driver and SQL, not a mock.
func setupPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
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

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `
		CREATE TABLE orders (
			id          UUID PRIMARY KEY,
			customer_id TEXT NOT NULL,
			item_sku    TEXT NOT NULL,
			quantity    INTEGER NOT NULL CHECK (quantity > 0),
			status      TEXT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL
		)
	`)
	require.NoError(t, err)

	return pool
}

func TestOrderRepository_SaveAndFindByID(t *testing.T) {
	pool := setupPostgres(t)
	repo := repository.NewOrderRepository(pool)
	ctx := context.Background()

	order := &domain.Order{
		ID:         uuid.NewString(),
		CustomerID: "cust-1",
		ItemSKU:    "sku-1",
		Quantity:   2,
		Status:     domain.StatusPending,
		CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}

	err := repo.Save(ctx, order)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, order.ID, found.ID)
	require.Equal(t, order.CustomerID, found.CustomerID)
	require.Equal(t, order.ItemSKU, found.ItemSKU)
	require.Equal(t, order.Quantity, found.Quantity)
	require.Equal(t, order.Status, found.Status)
	require.WithinDuration(t, order.CreatedAt, found.CreatedAt, time.Second)
}

func TestOrderRepository_FindByID_NotFound(t *testing.T) {
	pool := setupPostgres(t)
	repo := repository.NewOrderRepository(pool)

	_, err := repo.FindByID(context.Background(), uuid.NewString())
	require.ErrorIs(t, err, repository.ErrOrderNotFound)
}

func TestOrderRepository_Save_DuplicateID(t *testing.T) {
	pool := setupPostgres(t)
	repo := repository.NewOrderRepository(pool)
	ctx := context.Background()

	order := &domain.Order{
		ID:         uuid.NewString(),
		CustomerID: "cust-1",
		ItemSKU:    "sku-1",
		Quantity:   1,
		Status:     domain.StatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	require.NoError(t, repo.Save(ctx, order))
	require.Error(t, repo.Save(ctx, order))
}
