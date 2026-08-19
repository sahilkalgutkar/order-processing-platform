package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/domain"
)

// OrderCreated is the wire contract published to SNS. Consumers
// (inventory-service, notification-service) depend on this shape,
// not on the order-service's internal domain type.
type OrderCreated struct {
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	ItemSKU    string    `json:"item_sku"`
	Quantity   int       `json:"quantity"`
	OccurredAt time.Time `json:"occurred_at"`
}

type Publisher interface {
	PublishOrderCreated(ctx context.Context, o *domain.Order) error
}

// SNSClient is the one method SNSPublisher actually calls, defined here
// rather than depending on the concrete *sns.Client — same
// dependency-inversion shape as OrderStore/EventPublisher at the HTTP
// layer. Lets the JSON body and message-attribute construction be
// unit-tested against a fake instead of needing a real (or LocalStack) SNS
// topic.
type SNSClient interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

type SNSPublisher struct {
	client   SNSClient
	topicArn string
}

func NewSNSPublisher(client SNSClient, topicArn string) *SNSPublisher {
	return &SNSPublisher{client: client, topicArn: topicArn}
}

func (p *SNSPublisher) PublishOrderCreated(ctx context.Context, o *domain.Order) error {
	event := OrderCreated{
		OrderID:    o.ID,
		CustomerID: o.CustomerID,
		ItemSKU:    o.ItemSKU,
		Quantity:   o.Quantity,
		OccurredAt: o.CreatedAt,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal order created event: %w", err)
	}

	_, err = p.client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.topicArn),
		Message:  aws.String(string(body)),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			"eventType": {
				DataType:    aws.String("String"),
				StringValue: aws.String("OrderCreated"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("publish order created event: %w", err)
	}
	return nil
}
