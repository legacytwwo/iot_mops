package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	"rule_engine/internal/entities"
)

const alertsCollection = "alerts"

type AlertsRepository struct {
	coll *mongo.Collection
}

func NewAlertsRepository(db *mongo.Database) *AlertsRepository {
	return &AlertsRepository{coll: db.Collection(alertsCollection)}
}

func (r *AlertsRepository) Insert(ctx context.Context, alert entities.Alert) error {
	if _, err := r.coll.InsertOne(ctx, alert); err != nil {
		return fmt.Errorf("insert alert: %w", err)
	}
	return nil
}

func (r *AlertsRepository) InsertMany(ctx context.Context, alerts []entities.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	docs := make([]interface{}, 0, len(alerts))
	for _, a := range alerts {
		docs = append(docs, a)
	}

	if _, err := r.coll.InsertMany(ctx, docs); err != nil {
		return fmt.Errorf("insert many alerts: %w", err)
	}
	return nil
}
