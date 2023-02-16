package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

type mongoHelper struct {
	ctx    *context.Context
	client *mongo.Client
	db     *mongo.Database
}

func NewMongoDBHelper(url, dbname string) NoSQLDBHelper {
	ctx := context.Background()
	client, db, err := initMongoDB(ctx, url, dbname)
	if err != nil {
		fmt.Println("Panic Failed to init mysql", zap.Error(err)) // not log
	}
	return &mongoHelper{
		ctx:    &ctx,
		client: client,
		db:     db,
	}
}

func initMongoDB(ctx context.Context, url, dbname string) (*mongo.Client, *mongo.Database, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(url))
	if err != nil {
		return nil, nil, err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, nil, err
	}

	dbName := client.Database(dbname)
	return client, dbName, nil
}

func (h *mongoHelper) Collection(name string) *mongo.Collection {
	return h.db.Collection(name)
}

func (h *mongoHelper) Close() error {
	return h.client.Disconnect(*h.ctx)
}
