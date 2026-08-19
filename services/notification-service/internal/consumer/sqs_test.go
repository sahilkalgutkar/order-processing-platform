package consumer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeOrderCreated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid payload",
			body:    `{"order_id":"o-1","customer_id":"c-1","item_sku":"sku-1","quantity":2,"occurred_at":"2026-01-01T00:00:00Z"}`,
			wantErr: false,
		},
		{
			name:    "malformed json is rejected",
			body:    `{not-json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event, err := decodeOrderCreated(tt.body)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "o-1", event.OrderID)
			assert.Equal(t, "c-1", event.CustomerID)
			assert.Equal(t, "sku-1", event.ItemSKU)
			assert.Equal(t, 2, event.Quantity)
		})
	}
}
