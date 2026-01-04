package mongo

import (
	"context"
	"fmt"
	"iot_controller/entities"
	"iot_controller/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	db         *mongo.Database
	eventColl  *mongo.Collection
	deviceColl *mongo.Collection
}

func New(db *mongo.Database, eventCollName, deviceCollName string) repository.EventRepository {
	return &mongoRepository{
		db:         db,
		eventColl:  db.Collection(eventCollName),
		deviceColl: db.Collection(deviceCollName),
	}
}

func (r *mongoRepository) SaveEvent(ctx context.Context, event *entities.Event) error {
	_, err := r.eventColl.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("mongo insert event error: %w", err)
	}
	return nil
}

func (r *mongoRepository) SaveDevice(ctx context.Context, device *entities.Device) error {
	filter := bson.M{"id": device.ID}
	update := bson.M{"$set": device}

	opts := options.UpdateOne().SetUpsert(true)

	_, err := r.deviceColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("mongo upsert device error: %w", err)
	}
	return nil
}
