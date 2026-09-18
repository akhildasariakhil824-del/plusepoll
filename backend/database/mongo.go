package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
	Users    *mongo.Collection
	Polls    *mongo.Collection
	Votes    *mongo.Collection
}

func InitMongoDB(uri, dbName string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	usersCol := db.Collection("users")
	pollsCol := db.Collection("polls")
	votesCol := db.Collection("votes")

	// Ensure unique index on user email
	_, err = usersCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Printf("Warning: failed creating user email index: %v", err)
	}

	// Ensure index on polls creator and created_at
	_, err = pollsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "creator_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	})
	if err != nil {
		log.Printf("Warning: failed creating poll indexes: %v", err)
	}

	// Ensure compound index on votes (poll_id, voter_id)
	_, err = votesCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "poll_id", Value: 1},
			{Key: "voter_id", Value: 1},
		},
	})
	if err != nil {
		log.Printf("Warning: failed creating vote index: %v", err)
	}

	log.Println(" Successfully connected to MongoDB and verified indexes")

	return &MongoDB{
		Client:   client,
		Database: db,
		Users:    usersCol,
		Polls:    pollsCol,
		Votes:    votesCol,
	}, nil
}
