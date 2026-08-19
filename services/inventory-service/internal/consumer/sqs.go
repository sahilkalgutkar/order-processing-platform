package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/domain"
	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/events"
	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/repository"
)

// StockLookup abstracts "how much of this SKU do we have." A real system
// would back this with a stock table; here it's an in-memory stand-in so
// the reservation flow (the part worth demonstrating) isn't gated on a
// second datastore.
type StockLookup interface {
	Available(sku string) int
}

type FixedStock struct {
	Default int
}

func (f FixedStock) Available(string) int { return f.Default }

type Consumer struct {
	sqsClient *sqs.Client
	queueURL  string
	repo      *repository.ReservationRepository
	stock     StockLookup
	logger    *slog.Logger
}

func New(sqsClient *sqs.Client, queueURL string, repo *repository.ReservationRepository, stock StockLookup, logger *slog.Logger) *Consumer {
	return &Consumer{sqsClient: sqsClient, queueURL: queueURL, repo: repo, stock: stock, logger: logger}
}

// Run long-polls SQS until ctx is canceled. The queue subscription is
// configured with RawMessageDelivery=true (see scripts/localstack-init.sh),
// so the message body is the OrderCreated JSON payload directly rather
// than an SNS notification envelope.
func (c *Consumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		out, err := c.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     10,
			VisibilityTimeout:   30,
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("receive message failed", "error", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, msg := range out.Messages {
			c.handle(ctx, msg)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, msg sqstypes.Message) {
	var event events.OrderCreated
	if err := json.Unmarshal([]byte(aws.ToString(msg.Body)), &event); err != nil {
		c.logger.Error("failed to unmarshal order created event, dropping", "error", err)
		c.delete(ctx, msg)
		return
	}

	available := c.stock.Available(event.ItemSKU)
	reservation, _ := domain.Reserve(event.OrderID, event.ItemSKU, event.Quantity, available, time.Now().UTC())

	if err := c.repo.Save(ctx, reservation); err != nil {
		c.logger.Error("failed to save reservation, message will be retried", "error", err, "order_id", event.OrderID)
		return // don't delete — let it become visible again and retry
	}

	c.logger.Info("reservation processed", "order_id", event.OrderID, "status", reservation.Status)
	c.delete(ctx, msg)
}

func (c *Consumer) delete(ctx context.Context, msg sqstypes.Message) {
	_, err := c.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		c.logger.Error("failed to delete message", "error", err)
	}
}
