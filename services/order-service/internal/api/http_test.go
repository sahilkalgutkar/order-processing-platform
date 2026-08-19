package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/api"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/domain"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/repository"
)

// fakeStore is an in-memory OrderStore so handler tests never touch Postgres.
type fakeStore struct {
	orders  map[string]*domain.Order
	saveErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{orders: make(map[string]*domain.Order)}
}

func (f *fakeStore) Save(_ context.Context, o *domain.Order) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.orders[o.ID] = o
	return nil
}

func (f *fakeStore) FindByID(_ context.Context, id string) (*domain.Order, error) {
	o, ok := f.orders[id]
	if !ok {
		return nil, repository.ErrOrderNotFound
	}
	return o, nil
}

// fakePublisher records published events instead of calling SNS.
type fakePublisher struct {
	published []*domain.Order
	err       error
}

func (f *fakePublisher) PublishOrderCreated(_ context.Context, o *domain.Order) error {
	if f.err != nil {
		return f.err
	}
	f.published = append(f.published, o)
	return nil
}

func newTestServer(store *fakeStore, pub *fakePublisher) *httptest.Server {
	h := api.NewHandler(store, pub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return httptest.NewServer(h.Routes())
}

func TestCreateOrder_Success(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	pub := &fakePublisher{}
	srv := newTestServer(store, pub)
	defer srv.Close()

	body := bytes.NewBufferString(`{"customer_id":"cust-1","item_sku":"sku-1","quantity":3}`)
	resp, err := http.Post(srv.URL+"/orders", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var created domain.Order
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	require.Equal(t, "cust-1", created.CustomerID)
	require.Equal(t, domain.StatusPending, created.Status)

	require.Len(t, store.orders, 1)
	require.Len(t, pub.published, 1)
	require.Equal(t, created.ID, pub.published[0].ID)
}

func TestCreateOrder_ValidationError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(newFakeStore(), &fakePublisher{})
	defer srv.Close()

	body := bytes.NewBufferString(`{"customer_id":"","item_sku":"sku-1","quantity":1}`)
	resp, err := http.Post(srv.URL+"/orders", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestCreateOrder_PublishFailureStillReturnsCreated(t *testing.T) {
	t.Parallel()

	// The handler treats "saved but not yet published" as acceptable —
	// the order is durable and a reconciliation job can re-publish it.
	// A publish error must not turn into a 500 for the caller.
	store := newFakeStore()
	pub := &fakePublisher{err: context.DeadlineExceeded}
	srv := newTestServer(store, pub)
	defer srv.Close()

	body := bytes.NewBufferString(`{"customer_id":"cust-1","item_sku":"sku-1","quantity":1}`)
	resp, err := http.Post(srv.URL+"/orders", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Len(t, store.orders, 1)
	require.Empty(t, pub.published)
}

func TestGetOrder_NotFound(t *testing.T) {
	t.Parallel()

	srv := newTestServer(newFakeStore(), &fakePublisher{})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/orders/does-not-exist")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetOrder_Found(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	order, err := domain.NewOrder("cust-1", "sku-1", 1)
	require.NoError(t, err)
	require.NoError(t, store.Save(context.Background(), order))

	srv := newTestServer(store, &fakePublisher{})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/orders/" + order.ID)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var got domain.Order
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, order.ID, got.ID)
}
