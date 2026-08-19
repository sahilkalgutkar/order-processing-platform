//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tcmongodb "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/domain"
	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/repository"
)

func setupMongo(t *testing.T) *mongo.Database {
	t.Helper()
	ctx := context.Background()

	container, err := tcmongodb.Run(ctx, "mongo:7")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })

	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Disconnect(context.Background())) })

	return client.Database("inventory")
}

func TestReservationRepository_Save(t *testing.T) {
	db := setupMongo(t)
	repo := repository.NewReservationRepository(db)
	ctx := context.Background()

	res := domain.Reservation{
		OrderID:    "order-1",
		ItemSKU:    "sku-1",
		Quantity:   2,
		Status:     domain.ReservationConfirmed,
		ReservedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	err := repo.Save(ctx, res)
	require.NoError(t, err)

	var found domain.Reservation
	err = db.Collection("reservations").FindOne(ctx, bson.M{"order_id": "order-1"}).Decode(&found)
	require.NoError(t, err)
	require.Equal(t, res.OrderID, found.OrderID)
	require.Equal(t, res.ItemSKU, found.ItemSKU)
	require.Equal(t, res.Quantity, found.Quantity)
	require.Equal(t, res.Status, found.Status)
	require.WithinDuration(t, res.ReservedAt, found.ReservedAt, time.Second)
}

func TestReservationRepository_Save_Backordered(t *testing.T) {
	db := setupMongo(t)
	repo := repository.NewReservationRepository(db)
	ctx := context.Background()

	res := domain.Reservation{
		OrderID:    "order-2",
		ItemSKU:    "sku-2",
		Quantity:   5,
		Status:     domain.ReservationBackordered,
		ReservedAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	require.NoError(t, repo.Save(ctx, res))

	var found domain.Reservation
	err := db.Collection("reservations").FindOne(ctx, bson.M{"order_id": "order-2"}).Decode(&found)
	require.NoError(t, err)
	require.Equal(t, domain.ReservationBackordered, found.Status)
}
