package mocks

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoCollection is a mock implementation of MongoCollectionInt.
// It allows testing of MongoDB operations without requiring a real database connection.
type MongoCollection struct {
	InsertOneFunc        func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	FindFunc             func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error)
	FindOneFunc          func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult
	FindOneAndUpdateFunc func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult
	DeleteOneFunc        func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	DeleteManyFunc       func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
}

// InsertOne mocks the InsertOne method of a MongoDB collection.
func (m *MongoCollection) InsertOne(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
	return m.InsertOneFunc(ctx, document)
}

// Find mocks the Find method of a MongoDB collection.
func (m *MongoCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return m.FindFunc(ctx, filter, opts...)
}

// FindOne mocks the FindOne method of a MongoDB collection.
func (m *MongoCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	return m.FindOneFunc(ctx, filter, opts...)
}

// FindOneAndUpdate mocks the FindOneAndUpdate method of a MongoDB collection.
func (m *MongoCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
	return m.FindOneAndUpdateFunc(ctx, filter, update, opts...)
}

// DeleteOne mocks the DeleteOne method of a MongoDB collection.
func (m *MongoCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return m.DeleteOneFunc(ctx, filter, opts...)
}

// DeleteMany mocks the DeleteMany method of a MongoDB collection.
func (m *MongoCollection) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return m.DeleteManyFunc(ctx, filter, opts...)
}

// MockCursor is a mock implementation of mongo.Cursor.
// It is used to simulate the behavior of a MongoDB cursor for testing purposes.
type MockCursor struct {
	*mongo.Cursor
}

// NewMockCursor creates a new MockCursor from a slice of interface{}.
// It simulates the behavior of a MongoDB cursor with the provided data.
func NewMockCursor(data []interface{}) *MockCursor {
	cursor, err := mongo.NewCursorFromDocuments(data, nil, nil)
	if err != nil {
		return nil
	}
	return &MockCursor{Cursor: cursor}
}
