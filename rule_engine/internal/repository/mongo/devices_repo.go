package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"rule_engine/internal/entities"
)

const devicesCollection = "devices"

type DevicesRepository struct {
	coll *mongo.Collection
}

func NewDevicesRepository(db *mongo.Database) *DevicesRepository {
	return &DevicesRepository{coll: db.Collection(devicesCollection)}
}

func (r *DevicesRepository) FindIDsByScope(ctx context.Context, scope entities.DeviceScope) ([]string, error) {
	filter := bson.M{}
	if len(scope.IDs) > 0 {
		filter["_id"] = bson.M{"$in": scope.IDs}
	}
	if scope.Type != "" {
		filter["type"] = scope.Type
	}
	if scope.Location != "" {
		filter["location"] = scope.Location
	}

	cur, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find devices: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()

	out := make([]string, 0)
	for cur.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode device: %w", err)
		}
		out = append(out, doc.ID)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iter devices: %w", err)
	}
	return out, nil
}
