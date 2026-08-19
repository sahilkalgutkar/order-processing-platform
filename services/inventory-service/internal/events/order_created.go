package events

import "time"

// OrderCreated mirrors the contract published by order-service
// (services/order-service/internal/events/publisher.go). It's duplicated
// rather than imported so the two services stay independently deployable —
// this is the tradeoff of not sharing a library across service boundaries,
// and in a larger system it's the kind of thing a proto/JSON-schema
// contract registry (see proto/order/v1/order.proto) keeps in sync.
type OrderCreated struct {
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	ItemSKU    string    `json:"item_sku"`
	Quantity   int       `json:"quantity"`
	OccurredAt time.Time `json:"occurred_at"`
}
