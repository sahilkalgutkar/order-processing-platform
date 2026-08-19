package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
)

var (
	ErrCustomerIDRequired = errors.New("customer id is required")
	ErrItemSKURequired    = errors.New("item sku is required")
	ErrQuantityInvalid    = errors.New("quantity must be greater than zero")
)

type Order struct {
	ID         string      `json:"id"`
	CustomerID string      `json:"customer_id"`
	ItemSKU    string      `json:"item_sku"`
	Quantity   int         `json:"quantity"`
	Status     OrderStatus `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
}

// NewOrder validates input and constructs a new pending Order.
// Validation lives here, not in the HTTP handler, so it is enforced
// regardless of transport (REST today, gRPC later).
func NewOrder(customerID, itemSKU string, quantity int) (*Order, error) {
	if customerID == "" {
		return nil, ErrCustomerIDRequired
	}
	if itemSKU == "" {
		return nil, ErrItemSKURequired
	}
	if quantity <= 0 {
		return nil, ErrQuantityInvalid
	}

	return &Order{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		ItemSKU:    itemSKU,
		Quantity:   quantity,
		Status:     StatusPending,
		CreatedAt:  time.Now().UTC(),
	}, nil
}
