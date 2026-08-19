package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/events"
	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/notifier"
)

type Consumer struct {
	sqsClient *sqs.Client
	queueURL  string
	notifier  notifier.Notifier
	logger    *slog.Logger
}

func New(sqsClient *sqs.Client, queueURL string, n notifier.Notifier, logger *slog.Logger) *Consumer {
	return &Consumer{sqsClient: sqsClient, queueURL: queueURL, notifier: n, logger: logger}
}

// Run long-polls its own SQS queue independently of inventory-service's
// queue — both are subscribed to the same SNS topic, so this is the
// fan-out half of the pub/sub demo: one OrderCreated event, two
// independent consumers, neither aware of the other.
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
	event, err := decodeOrderCreated(aws.ToString(msg.Body))
	if err != nil {
		c.logger.Error("failed to unmarshal order created event, dropping", "error", err)
		c.delete(ctx, msg)
		return
	}

	if err := c.notifier.Notify(ctx, event); err != nil {
		c.logger.Error("notify failed, message will be retried", "error", err, "order_id", event.OrderID)
		return
	}

	c.delete(ctx, msg)
}

// decodeOrderCreated is split out from handle so the message-parsing
// logic can be unit tested without a real SQS client or message.
func decodeOrderCreated(body string) (events.OrderCreated, error) {
	var event events.OrderCreated
	err := json.Unmarshal([]byte(body), &event)
	return event, err
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
