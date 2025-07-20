package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aph138/dekamond/internal/entity"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	UserCollection = "user"
)

// MyMongo defines a helper struct for connecting to mongodb database
type MyMongo struct {
	db      *mongo.Database
	timeout time.Duration
}

// Timeout is used as a global timeout for all of the operations for more convenience.
// In a real world scenario each operation should have its own timeout.
func NewMongo(address, name string, timeout time.Duration, opt *options.ClientOptions) (*MyMongo, error) {
	if opt == nil {
		opt = options.Client().ApplyURI(address)
	} else {
		opt.ApplyURI(address)
	}
	client, err := mongo.Connect(opt)
	if err != nil {
		return nil, fmt.Errorf("err when connecting to db at %s: %w", address, err)
	}

	// check for connection
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("err when pinging db: %w", err)
	}
	db := client.Database(name)
	if err := createIndices(db); err != nil {
		return nil, fmt.Errorf("err when creating indices: %w", err)
	}
	return &MyMongo{
		db:      db,
		timeout: timeout,
	}, nil
}

// create index for phone field to improving performance
func createIndices(db *mongo.Database) error {
	userIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "phone", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := db.Collection(UserCollection).Indexes().CreateOne(context.Background(), userIndexModel)
	if err != nil {
		return fmt.Errorf("err when creating user index: %w", err)
	}
	return nil
}
func (d *MyMongo) InsertOne(col string, doc any, opts ...options.Lister[options.InsertOneOptions]) (*bson.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	result, err := d.db.Collection(col).InsertOne(ctx, doc, opts...)
	if err != nil {
		return nil, fmt.Errorf("err when inserting one to %s: %w", col, err)
	}
	id := result.InsertedID.(*bson.ObjectID)
	return id, nil
}
func (d *MyMongo) FindOne(col string, filter, output any, opts ...options.Lister[options.FindOneOptions]) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	if err := d.db.Collection(col).FindOne(ctx, filter, opts...).Decode(output); err != nil {
		return fmt.Errorf("err when finding one from %s: %w", col, err)
	}
	return nil
}

func (d *MyMongo) Count(col string, filter any, opts ...options.Lister[options.CountOptions]) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	number, err := d.db.Collection(col).CountDocuments(ctx, filter, opts...)
	if err != nil {
		return 0, fmt.Errorf("err when counting docs from %s: %w", col, err)
	}
	return number, nil
}
func (d *MyMongo) UpdateOne(col string, filter, query any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	return d.db.Collection(col).UpdateOne(ctx, filter, query, opts...)

}

func (d *MyMongo) SaveUser(phone string) (string, error) {
	filter := bson.M{"phone": phone}
	upsertQuery := bson.M{"$setOnInsert": entity.User{Phone: phone}}
	result, err := d.UpdateOne(UserCollection, filter, upsertQuery, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return "", fmt.Errorf("err when upserting user with mongodb: %w", err)
	}
	if result.UpsertedCount > 0 {
		return result.UpsertedID.(bson.ObjectID).Hex(), nil
	} else {
		var user entity.User
		err := d.FindOne(UserCollection, bson.M{"phone": phone}, &user)
		if err != nil {
			return "", fmt.Errorf("err when finding user in save method with mongodb: %w", err)
		}
		return user.ID.Hex(), nil
	}
}
