package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"rule_engine/internal/entities"
)

const rulesCollection = "rules"

type RulesRepository struct {
	coll *mongo.Collection
}

func NewRulesRepository(db *mongo.Database) *RulesRepository {
	return &RulesRepository{coll: db.Collection(rulesCollection)}
}

func (r *RulesRepository) FindEnabledByMetric(ctx context.Context, metric string) ([]entities.Rule, error) {
	filter := bson.M{
		"enabled":       true,
		"scope.metrics": metric,
	}

	cur, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find rules: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()

	var out []entities.Rule
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode rules: %w", err)
	}
	return out, nil
}

func (r *RulesRepository) FindInactivityRules(ctx context.Context) ([]entities.Rule, error) {
	filter := bson.M{
		"enabled":         true,
		"evaluation.type": entities.EvaluationInactivity,
	}

	cur, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find inactivity rules: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()

	var out []entities.Rule
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode inactivity rules: %w", err)
	}
	return out, nil
}
