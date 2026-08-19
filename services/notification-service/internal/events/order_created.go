package events

import "time"

// OrderCreated mirrors the contract published by order-service. See the
// identical comment in inventory-service for why it's duplicated rather
// than shared.
type OrderCreated struct {
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	ItemSKU    string    `json:"item_sku"`
	Quantity   int       `json:"quantity"`
	OccurredAt time.Time `json:"occurred_at"`
}
