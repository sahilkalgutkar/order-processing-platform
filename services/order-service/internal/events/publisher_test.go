package events_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/domain"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/events"
)

// fakeSNSClient is a hand-rolled test double for the SNSClient interface —
// no LocalStack, no network — matching the fakes used for SQS elsewhere in
// this repo.
type fakeSNSClient struct {
	err          error
	publishCalls []*sns.PublishInput
}

func (f *fakeSNSClient) Publish(_ context.Context, params *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
	f.publishCalls = append(f.publishCalls, params)
	if f.err != nil {
		return nil, f.err
	}
	return &sns.PublishOutput{}, nil
}

func TestSNSPublisher_PublishOrderCreated_Success(t *testing.T) {
	fake := &fakeSNSClient{}
	p := events.NewSNSPublisher(fake, "arn:aws:sns:us-east-1:000000000000:order-events")

	order := &domain.Order{
		ID:         "order-1",
		CustomerID: "cust-1",
		ItemSKU:    "sku-1",
		Quantity:   2,
		Status:     domain.StatusPending,
		CreatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	err := p.PublishOrderCreated(context.Background(), order)

	require.NoError(t, err)
	require.Len(t, fake.publishCalls, 1)

	call := fake.publishCalls[0]
	assert.Equal(t, "arn:aws:sns:us-east-1:000000000000:order-events", aws.ToString(call.TopicArn))
	assert.Equal(t, "String", aws.ToString(call.MessageAttributes["eventType"].DataType))
	assert.Equal(t, "OrderCreated", aws.ToString(call.MessageAttributes["eventType"].StringValue))

	var published events.OrderCreated
	require.NoError(t, json.Unmarshal([]byte(aws.ToString(call.Message)), &published))
	assert.Equal(t, "order-1", published.OrderID)
	assert.Equal(t, "cust-1", published.CustomerID)
	assert.Equal(t, "sku-1", published.ItemSKU)
	assert.Equal(t, 2, published.Quantity)
	assert.True(t, order.CreatedAt.Equal(published.OccurredAt))
}

func TestSNSPublisher_PublishOrderCreated_PublishError(t *testing.T) {
	fake := &fakeSNSClient{err: errors.New("sns unavailable")}
	p := events.NewSNSPublisher(fake, "arn:aws:sns:us-east-1:000000000000:order-events")

	order := &domain.Order{ID: "order-1", CustomerID: "cust-1", ItemSKU: "sku-1", Quantity: 1, CreatedAt: time.Now()}

	err := p.PublishOrderCreated(context.Background(), order)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish order created event")
}
