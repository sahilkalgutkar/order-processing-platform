package domain

import "time"

type ReservationStatus string

const (
	ReservationConfirmed   ReservationStatus = "confirmed"
	ReservationBackordered ReservationStatus = "backordered"
)

type Reservation struct {
	OrderID    string            `bson:"order_id"`
	ItemSKU    string            `bson:"item_sku"`
	Quantity   int               `bson:"quantity"`
	Status     ReservationStatus `bson:"status"`
	ReservedAt time.Time         `bson:"reserved_at"`
}

// Reserve applies the reservation rule against current stock: fulfill in
// full if enough stock is on hand, otherwise backorder the whole line
// rather than partially fulfilling it. Kept pure and separate from the
// SQS/Mongo plumbing so it can be table-tested without any infrastructure.
func Reserve(orderID, itemSKU string, requested, available int, now time.Time) (Reservation, int) {
	if requested <= available {
		return Reservation{
			OrderID:    orderID,
			ItemSKU:    itemSKU,
			Quantity:   requested,
			Status:     ReservationConfirmed,
			ReservedAt: now,
		}, available - requested
	}

	return Reservation{
		OrderID:    orderID,
		ItemSKU:    itemSKU,
		Quantity:   requested,
		Status:     ReservationBackordered,
		ReservedAt: now,
	}, available
}
