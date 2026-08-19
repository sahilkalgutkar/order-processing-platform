package notifier

import (
	"context"
	"log/slog"

	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/events"
)

// Notifier abstracts the delivery channel so the consumer loop doesn't
// care whether notifications go out over email, SMS, or (as here) a log
// line. Swapping in a real provider means implementing this interface,
// not touching the consumer.
type Notifier interface {
	Notify(ctx context.Context, event events.OrderCreated) error
}

type LogNotifier struct {
	logger *slog.Logger
}

func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	return &LogNotifier{logger: logger}
}

func (n *LogNotifier) Notify(_ context.Context, event events.OrderCreated) error {
	n.logger.Info("notification sent",
		"channel", "email",
		"customer_id", event.CustomerID,
		"order_id", event.OrderID,
		"item_sku", event.ItemSKU,
	)
	return nil
}
