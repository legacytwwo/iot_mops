package mongo

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Migration struct {
	ID string
	Up func(ctx context.Context, db *mongo.Database) error
}

var migrations = []Migration{
	{
		ID: "2026-01-05-01_rules_indexes",
		Up: func(ctx context.Context, db *mongo.Database) error {
			coll := db.Collection("rules")
			models := []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "enabled", Value: 1}, {Key: "scope.metrics", Value: 1}},
					Options: options.Index().SetName("rules_enabled_metrics"),
				},
				{
					Keys:    bson.D{{Key: "enabled", Value: 1}, {Key: "evaluation.type", Value: 1}},
					Options: options.Index().SetName("rules_enabled_eval_type"),
				},
			}
			_, err := coll.Indexes().CreateMany(ctx, models)
			return err
		},
	},
	{
		ID: "2026-01-05-02_alerts_indexes",
		Up: func(ctx context.Context, db *mongo.Database) error {
			coll := db.Collection("alerts")
			models := []mongo.IndexModel{
				{
					Keys: bson.D{{Key: "message_id", Value: 1}},
					Options: options.Index().
						SetName("alerts_message_id_unique").
						SetUnique(true).
						SetPartialFilterExpression(bson.M{"message_id": bson.M{"$exists": true, "$type": "string", "$gt": ""}}),
				},
			}
			_, err := coll.Indexes().CreateMany(ctx, models)
			return err
		},
	},
}

func Run(ctx context.Context, db *mongo.Database) error {
	applied, err := appliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.ID] {
			continue
		}
		if err := m.Up(ctx, db); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.ID, err)
		}
		if err := markApplied(ctx, db, m.ID); err != nil {
			return fmt.Errorf("mark applied %s: %w", m.ID, err)
		}
	}
	return nil
}

func AcquireLock(ctx context.Context, db *mongo.Database, ttl time.Duration) (string, bool, error) {
	coll := db.Collection("migration_lock")
	now := time.Now().UTC()
	until := now.Add(ttl)
	owner := lockOwner()

	filter := bson.M{
		"_id":          "mongo_migrations_lock",
		"locked_until": bson.M{"$lt": now},
	}
	update := bson.M{
		"$set": bson.M{
			"locked_at":    now,
			"locked_until": until,
			"owner":        owner,
		},
	}

	res := coll.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After))
	if res.Err() == nil {
		return owner, true, nil
	}
	if !errorsIsNoDocuments(res.Err()) {
		return "", false, fmt.Errorf("lock update: %w", res.Err())
	}

	_, err := coll.InsertOne(ctx, bson.M{
		"_id":          "mongo_migrations_lock",
		"locked_at":    now,
		"locked_until": until,
		"owner":        owner,
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("lock insert: %w", err)
	}
	return owner, true, nil
}

func ReleaseLock(ctx context.Context, db *mongo.Database, owner string) error {
	coll := db.Collection("migration_lock")
	_, err := coll.DeleteOne(ctx, bson.M{"_id": "mongo_migrations_lock", "owner": owner})
	if err != nil {
		return fmt.Errorf("release lock: %w", err)
	}
	return nil
}

func appliedMigrations(ctx context.Context, db *mongo.Database) (map[string]bool, error) {
	coll := db.Collection("migrations")
	cur, err := coll.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("find migrations: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()

	out := make(map[string]bool)
	for cur.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode migration: %w", err)
		}
		out[doc.ID] = true
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}
	return out, nil
}

func markApplied(ctx context.Context, db *mongo.Database, id string) error {
	coll := db.Collection("migrations")
	_, err := coll.InsertOne(ctx, bson.M{
		"_id":        id,
		"applied_at": time.Now().UTC(),
	})
	return err
}

func lockOwner() string {
	host, _ := os.Hostname()
	pid := strconv.Itoa(os.Getpid())
	if host == "" {
		host = "unknown"
	}
	return strings.Join([]string{host, pid}, ":")
}

func errorsIsNoDocuments(err error) bool {
	return err == mongo.ErrNoDocuments
}
