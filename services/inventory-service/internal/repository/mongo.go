package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/domain"
)

type ReservationRepository struct {
	collection *mongo.Collection
}

func NewReservationRepository(db *mongo.Database) *ReservationRepository {
	return &ReservationRepository{collection: db.Collection("reservations")}
}

func (r *ReservationRepository) Save(ctx context.Context, res domain.Reservation) error {
	_, err := r.collection.InsertOne(ctx, res)
	if err != nil {
		return fmt.Errorf("save reservation: %w", err)
	}
	return nil
}
