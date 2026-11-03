// Package mongostore contains generic MongoDB CRUD primitives. Domain adapters
// translate driver-level outcomes such as ErrNoDocuments into domain errors.
package mongostore

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Get[T any](ctx context.Context, collection *mongo.Collection, filter any) (T, error) {
	var value T
	err := collection.FindOne(ctx, filter).Decode(&value)
	return value, err
}

func Insert[T any](ctx context.Context, collection *mongo.Collection, value T) error {
	_, err := collection.InsertOne(ctx, value)
	return err
}

func Replace[T any](ctx context.Context, collection *mongo.Collection, filter any, value T) (bool, error) {
	result, err := collection.ReplaceOne(ctx, filter, value)
	if err != nil {
		return false, err
	}
	return result.MatchedCount > 0, nil
}

func List[T any](ctx context.Context, collection *mongo.Collection, filter any, findOptions *options.FindOptions) ([]T, error) {
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var values []T
	if err := cursor.All(ctx, &values); err != nil {
		return nil, err
	}
	return values, nil
}
