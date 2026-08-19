package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/domain"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Save(ctx context.Context, o *domain.Order) error {
	const q = `
		INSERT INTO orders (id, customer_id, item_sku, quantity, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, q, o.ID, o.CustomerID, o.ItemSKU, o.Quantity, o.Status, o.CreatedAt)
	if err != nil {
		return fmt.Errorf("save order: %w", err)
	}
	return nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	const q = `
		SELECT id, customer_id, item_sku, quantity, status, created_at
		FROM orders
		WHERE id = $1
	`
	var o domain.Order
	err := r.pool.QueryRow(ctx, q, id).Scan(&o.ID, &o.CustomerID, &o.ItemSKU, &o.Quantity, &o.Status, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find order: %w", err)
	}
	return &o, nil
}
