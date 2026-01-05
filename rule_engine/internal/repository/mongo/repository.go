package mongo

import "go.mongodb.org/mongo-driver/mongo"

type Repo struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Repo {
	return &Repo{db: db}
}
