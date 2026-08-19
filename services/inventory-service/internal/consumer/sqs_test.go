package consumer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/domain"
)

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

type fakeReservationSaver struct {
	err   error
	saved []domain.Reservation
}

func (f *fakeReservationSaver) Save(_ context.Context, res domain.Reservation) error {
	f.saved = append(f.saved, res)
	return f.err
}

func newTestConsumer(sqsClient *fakeSQSClient, repo *fakeReservationSaver, available int) *Consumer {
	return New(sqsClient, "queue-url", repo, FixedStock{Default: available}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func orderCreatedBody(quantity int) string {
	return fmt.Sprintf(`{"order_id":"o-1","customer_id":"c-1","item_sku":"sku-1","quantity":%d,"occurred_at":"2026-01-01T00:00:00Z"}`, quantity)
}

func TestHandle_SufficientStock_ConfirmsAndDeletes(t *testing.T) {
	fake := &fakeSQSClient{}
	repo := &fakeReservationSaver{}
	c := newTestConsumer(fake, repo, 10)

	msg := sqstypes.Message{Body: aws.String(orderCreatedBody(2)), ReceiptHandle: aws.String("receipt-1")}
	c.handle(context.Background(), msg)

	require.Len(t, repo.saved, 1)
	assert.Equal(t, domain.ReservationConfirmed, repo.saved[0].Status)
	assert.Equal(t, "o-1", repo.saved[0].OrderID)
	assert.Contains(t, fake.deletedReceiptHandles, "receipt-1")
}

func TestHandle_InsufficientStock_Backorders(t *testing.T) {
	fake := &fakeSQSClient{}
	repo := &fakeReservationSaver{}
	c := newTestConsumer(fake, repo, 1)

	msg := sqstypes.Message{Body: aws.String(orderCreatedBody(5)), ReceiptHandle: aws.String("receipt-1")}
	c.handle(context.Background(), msg)

	require.Len(t, repo.saved, 1)
	assert.Equal(t, domain.ReservationBackordered, repo.saved[0].Status)
	assert.Contains(t, fake.deletedReceiptHandles, "receipt-1")
}

func TestHandle_MalformedJSON_DropsWithoutSaving(t *testing.T) {
	fake := &fakeSQSClient{}
	repo := &fakeReservationSaver{}
	c := newTestConsumer(fake, repo, 10)

	msg := sqstypes.Message{Body: aws.String(`{not-json`), ReceiptHandle: aws.String("receipt-1")}
	c.handle(context.Background(), msg)

	assert.Empty(t, repo.saved)
	assert.Contains(t, fake.deletedReceiptHandles, "receipt-1")
}

func TestHandle_SaveFails_LeavesMessageForRetry(t *testing.T) {
	fake := &fakeSQSClient{}
	repo := &fakeReservationSaver{err: errors.New("mongo down")}
	c := newTestConsumer(fake, repo, 10)

	msg := sqstypes.Message{Body: aws.String(orderCreatedBody(2)), ReceiptHandle: aws.String("receipt-1")}
	c.handle(context.Background(), msg)

	require.Len(t, repo.saved, 1)
	assert.Empty(t, fake.deletedReceiptHandles)
}

func TestRun_ProcessesMessagesThenExitsOnContextCancel(t *testing.T) {
	fake := &fakeSQSClient{
		receiveResponses: []*sqs.ReceiveMessageOutput{
			{Messages: []sqstypes.Message{
				{Body: aws.String(orderCreatedBody(2)), ReceiptHandle: aws.String("receipt-1")},
			}},
		},
	}
	repo := &fakeReservationSaver{}
	c := newTestConsumer(fake, repo, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	c.Run(ctx)

	require.Len(t, repo.saved, 1)
	assert.Contains(t, fake.deletedReceiptHandles, "receipt-1")
}

func TestRun_ExitsImmediatelyOnAlreadyCanceledContext(t *testing.T) {
	fake := &fakeSQSClient{}
	repo := &fakeReservationSaver{}
	c := newTestConsumer(fake, repo, 10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c.Run(ctx)

	assert.Zero(t, fake.receiveCalls)
}
