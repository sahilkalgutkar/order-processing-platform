package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/domain"
)

func TestReserve(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		requested     int
		available     int
		wantStatus    domain.ReservationStatus
		wantRemaining int
	}{
		{
			name:          "enough stock is confirmed and decremented",
			requested:     3,
			available:     10,
			wantStatus:    domain.ReservationConfirmed,
			wantRemaining: 7,
		},
		{
			name:          "exact stock match is confirmed",
			requested:     10,
			available:     10,
			wantStatus:    domain.ReservationConfirmed,
			wantRemaining: 0,
		},
		{
			name:          "insufficient stock is backordered and untouched",
			requested:     11,
			available:     10,
			wantStatus:    domain.ReservationBackordered,
			wantRemaining: 10,
		},
		{
			name:          "zero stock is backordered",
			requested:     1,
			available:     0,
			wantStatus:    domain.ReservationBackordered,
			wantRemaining: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reservation, remaining := domain.Reserve("order-1", "sku-1", tt.requested, tt.available, now)

			assert.Equal(t, tt.wantStatus, reservation.Status)
			assert.Equal(t, tt.requested, reservation.Quantity)
			assert.Equal(t, "order-1", reservation.OrderID)
			assert.Equal(t, now, reservation.ReservedAt)
			assert.Equal(t, tt.wantRemaining, remaining)
		})
	}
}
