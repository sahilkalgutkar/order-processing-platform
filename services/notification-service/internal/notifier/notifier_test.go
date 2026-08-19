package notifier_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/events"
	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/notifier"
)

func TestLogNotifier_Notify(t *testing.T) {
	t.Parallel()

	n := notifier.NewLogNotifier(slog.New(slog.NewTextHandler(io.Discard, nil)))

	event := events.OrderCreated{
		OrderID:    "order-1",
		CustomerID: "cust-1",
		ItemSKU:    "sku-1",
		Quantity:   1,
		OccurredAt: time.Now().UTC(),
	}

	err := n.Notify(context.Background(), event)
	require.NoError(t, err)
}
