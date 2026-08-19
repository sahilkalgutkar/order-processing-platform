package consumer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/events"
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

// fakeSQSClient is a hand-rolled test double for the SQSClient interface —
// no LocalStack, no network — matching how order-service's handler tests
// run against in-memory fakes for OrderStore/EventPublisher.
type fakeSQSClient struct {
	receiveResponses []*sqs.ReceiveMessageOutput
	receiveErr       error
	receiveCalls     int

	deletedReceiptHandles []string
	deleteErr             error
}

func (f *fakeSQSClient) ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if f.receiveErr != nil {
		return nil, f.receiveErr
	}
	if f.receiveCalls >= len(f.receiveResponses) {
		return &sqs.ReceiveMessageOutput{}, nil
	}
	out := f.receiveResponses[f.receiveCalls]
	f.receiveCalls++
	return out, nil
}

func (f *fakeSQSClient) DeleteMessage(_ context.Context, params *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	f.deletedReceiptHandles = append(f.deletedReceiptHandles, aws.ToString(params.ReceiptHandle))
	return &sqs.DeleteMessageOutput{}, f.deleteErr
}

type fakeNotifier struct {
	err      error
	notified []events.OrderCreated
}

func (f *fakeNotifier) Notify(_ context.Context, event events.OrderCreated) error {
	f.notified = append(f.notified, event)
	return f.err
}

func newTestConsumer(sqsClient *fakeSQSClient, n *fakeNotifier) *Consumer {
	return New(sqsClient, "queue-url", n, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

const validBody = `{"order_id":"o-1","customer_id":"c-1","item_sku":"sku-1","quantity":2,"occurred_at":"2026-01-01T00:00:00Z"}`

func TestHandle_Success_NotifiesAndDeletes(t *testing.T) {
	fake := &fakeSQSClient{}
	n := &fakeNotifier{}
	c := newTestConsumer(fake, n)

	msg := sqstypes.Message{Body: aws.String(validBody), ReceiptHandle: aws.String("receipt-1")}
	c.handle(context.Background(), msg)

	require.Len(t, n.notified, 1)
	assert.Equal(t, "o-1", n.notified[0].OrderID)
	assert.Contains(t, fake.deletedReceiptHandles, "receipt-1")
}

func TestHandle_MalformedJSON_DropsWithoutNotifying(t *testing.T) {
	fake := &fakeSQSClient{}
	n := &fakeNotifier{}
	c := newTestConsumer(fake, n)

	msg := sqstypes.Message{Body: aws.String(`{not-json`), ReceiptHandle: aws.String("receipt-1")}
	c.handle(context.Background(), msg)

	assert.Empty(t, n.notified)
	assert.Contains(t, fake.deletedReceiptHandles, "receipt-1")
}

func TestHandle_NotifyFails_LeavesMessageForRetry(t *testing.T) {
	fake := &fakeSQSClient{}
	n := &fakeNotifier{err: errors.New("smtp down")}
	c := newTestConsumer(fake, n)

	msg := sqstypes.Message{Body: aws.String(validBody), ReceiptHandle: aws.String("receipt-1")}
	c.handle(context.Background(), msg)

	require.Len(t, n.notified, 1)
	assert.Empty(t, fake.deletedReceiptHandles)
}

func TestRun_ProcessesMessagesThenExitsOnContextCancel(t *testing.T) {
	fake := &fakeSQSClient{
		receiveResponses: []*sqs.ReceiveMessageOutput{
			{Messages: []sqstypes.Message{
				{Body: aws.String(validBody), ReceiptHandle: aws.String("receipt-1")},
			}},
		},
	}
	n := &fakeNotifier{}
	c := newTestConsumer(fake, n)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	c.Run(ctx)

	require.Len(t, n.notified, 1)
	assert.Contains(t, fake.deletedReceiptHandles, "receipt-1")
}

func TestRun_ExitsImmediatelyOnAlreadyCanceledContext(t *testing.T) {
	fake := &fakeSQSClient{}
	n := &fakeNotifier{}
	c := newTestConsumer(fake, n)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c.Run(ctx)

	assert.Zero(t, fake.receiveCalls)
}
