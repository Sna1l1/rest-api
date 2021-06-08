package apiserver

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Storage struct {
	client *mongo.Client
	ctx    context.Context
	//cancel context.CancelFunc
}

func Init(url string) (*Storage, error) {
	ctx := context.TODO()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb+srv://snail:tim262373@cluster0.hdaxf.mongodb.net/sample_analytics?retryWrites=true&w=majority"))

	if err != nil {
		//cancel()
		return nil, err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		//cancel()
		return nil, err
	}
	return &Storage{
		client: client,
		ctx:    ctx,
		//cancel: cancel,
	}, nil
}

func InsertOne(client *mongo.Client, collection string, ctx context.Context, data interface{}) error {
	_, err := client.Database("sample_analytics").Collection(collection).InsertOne(ctx, data)
	if err != nil {
		return err
	}
	return nil
}
