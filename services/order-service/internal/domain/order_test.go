package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/domain"
)

func TestNewOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		customerID string
		itemSKU    string
		quantity   int
		wantErr    error
	}{
		{
			name:       "valid order",
			customerID: "cust-1",
			itemSKU:    "sku-1",
			quantity:   2,
			wantErr:    nil,
		},
		{
			name:       "missing customer id",
			customerID: "",
			itemSKU:    "sku-1",
			quantity:   2,
			wantErr:    domain.ErrCustomerIDRequired,
		},
		{
			name:       "missing item sku",
			customerID: "cust-1",
			itemSKU:    "",
			quantity:   2,
			wantErr:    domain.ErrItemSKURequired,
		},
		{
			name:       "zero quantity",
			customerID: "cust-1",
			itemSKU:    "sku-1",
			quantity:   0,
			wantErr:    domain.ErrQuantityInvalid,
		},
		{
			name:       "negative quantity",
			customerID: "cust-1",
			itemSKU:    "sku-1",
			quantity:   -5,
			wantErr:    domain.ErrQuantityInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order, err := domain.NewOrder(tt.customerID, tt.itemSKU, tt.quantity)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, order)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, order)
			assert.NotEmpty(t, order.ID)
			assert.Equal(t, tt.customerID, order.CustomerID)
			assert.Equal(t, tt.itemSKU, order.ItemSKU)
			assert.Equal(t, tt.quantity, order.Quantity)
			assert.Equal(t, domain.StatusPending, order.Status)
			assert.False(t, order.CreatedAt.IsZero())
		})
	}
}
