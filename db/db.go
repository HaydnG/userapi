package db

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"userapi/cacheStore"
	"userapi/data"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoCollectionInt is an interface that abstracts MongoDB operations.
type MongoCollectionInt interface {
	InsertOne(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error)
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error)
	FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult
	FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult
	DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
}

var client *mongo.Client
var userCollection MongoCollectionInt

// SetCollection allows setting a different MongoCollection, useful for testing.
func SetCollection(collection MongoCollectionInt) {
	userCollection = collection
}

// Init initializes the MongoDB driver and connection
func Init() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("attempting to connect to mongoDB at mongodb://localhost:27017")
	// I wouldn't typically suggest connecting to the database directly, since its harder to protect, as well as other limitations.
	// Due to the scale of this project, im sure its ok ;)
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return fmt.Errorf("failed to connect to mongoDB: %v, ensure the docker image has been ran", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to ping to mongoDB: %v, ensure the docker image has been ran", err)
	}
	log.Printf("successfully connected to mongoDB")

	userCollection = &MongoCollection{
		collection: client.Database("faceit").Collection("users"),
	}

	return nil
}

// GetUser queries the user by username, this is needed for check for duplicates on new user creation.
func GetUser(nickname string) (*data.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"nickname": nickname}
	var user data.User
	err := userCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}

	return &user, nil
}

// GetUserByID queries user by ID, ID will be indexed. So quicker to search
func GetUserByID(id string) (*data.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"_id": id}
	var user data.User
	err := userCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}

	return &user, nil
}

var userStore = cacheStore.NewStore[int, []data.User]("userStore", time.Second*20)

// GetUsers queries the database to get ALL the users
// Utilised a cache to reduce database hits
// Cache lifetime is 20 seconds. It'll be missing recent users, but nessesary for large scale systems to protect database performance.
// This is also where a softExpiry cache can be usefull.
func GetUsers() ([]data.User, error) {

	// just key on 0, we're not using this cache for anything complex
	users, err := userStore.GetData(0, func(key int) ([]data.User, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := userCollection.Find(ctx, bson.M{})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)

		// pre-alloc everying in a single call
		users := make([]data.User, 0, cursor.RemainingBatchLength())
		for cursor.Next(ctx) {
			var user data.User
			if err := cursor.Decode(&user); err != nil {
				return nil, err
			}
			users = append(users, user)
		}

		return users, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed when getting users: %v", err)
	}

	return users, nil

}

// GetUsersFiltered queries the database to find users matching the given query
func GetUsersFiltered(country, nickname string, createdAfter time.Time, page, pageSize int) ([]data.User, error) {

	filter := bson.M{}
	if country != "" {
		// country wild carded filter.
		filter["country"] = bson.M{"$regex": regexp.QuoteMeta(country), "$options": "i"}
	}
	if nickname != "" {
		// nickname wild carded filter.
		filter["nickname"] = bson.M{"$regex": regexp.QuoteMeta(nickname), "$options": "i"}
	}
	if !createdAfter.IsZero() {
		// $gt meaing Greater than, find fields greater than the given value
		filter["created_at"] = bson.M{"$gt": createdAfter}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	findOptions := options.Find()

	// Where should we start searching from
	findOptions.SetSkip(int64((page - 1) * pageSize))
	// How many to fetch
	findOptions.SetLimit(int64(pageSize))

	cursor, err := userCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// pre-alloc everying in a single call
	users := make([]data.User, 0, cursor.RemainingBatchLength())
	for cursor.Next(ctx) {
		var user data.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// InsertUser adds the given user to the database
func InsertUser(user *data.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := userCollection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("err when inserting user - err: %v", err)
	}

	return nil
}

// UpdateUser updates the given user's details in the database
func UpdateUser(user *data.User) (*data.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set the UpdatedAt field
	user.UpdatedAt = time.Now()

	// Create the update document
	update := bson.M{
		"$set": bson.M{
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"nickname":   user.Nickname,
			"password":   user.Password,
			"email":      user.Email,
			"country":    user.Country,
			"updated_at": user.UpdatedAt,
		},
	}

	// Options to return the updated document
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	// Perform the update operation
	var updatedUser data.User
	err := userCollection.FindOneAndUpdate(ctx, bson.M{"_id": user.ID}, update, opts).Decode(&updatedUser)
	if err != nil {
		return nil, fmt.Errorf("error when updating user - err: %v", err)
	}

	return &updatedUser, nil
}

// DeleteUser deletes the user with the given ID from the database
func DeleteUser(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create the filter to find the user by ID
	filter := bson.M{"_id": userID}

	// Perform the delete operation
	result, err := userCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("error when deleting user - err: %v", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("no user found with the given ID")
	}

	return nil
}

// DeleteAllUsers deletes all users from the database.
func DeleteAllUsers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Empty filter matches all
	filter := bson.M{}

	// Perform the delete operation.
	_, err := userCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting all users: %v", err)
	}

	return nil
}
