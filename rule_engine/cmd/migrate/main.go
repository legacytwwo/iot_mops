package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"rule_engine/internal/config"
	mongomigrate "rule_engine/internal/migrations/mongo"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Mongo.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("mongo ping: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	db := client.Database(cfg.Mongo.Database)
	owner, ok, err := mongomigrate.AcquireLock(ctx, db, cfg.Mongo.MigrateLockTTL)
	if err != nil {
		log.Fatalf("acquire lock: %v", err)
	}
	if !ok {
		log.Printf("migration lock is held by another process, exiting")
		return
	}
	defer func() {
		_ = mongomigrate.ReleaseLock(context.Background(), db, owner)
	}()

	if err := mongomigrate.Run(ctx, db); err != nil {
		log.Printf("migrations: %v", err)
		return
	}

	log.Printf("migrations applied")
}
