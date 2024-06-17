package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"userapi/data"
	"userapi/db"
	"userapi/mocks"
	"userapi/pb"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var grpcTestServer = &UserService{}

func init() {
	// Prevent all the error logs from showing up, even when expected
	// Comment this out if need to debug any issues
	// log.Default().SetOutput(io.Discard)
}

//################################################################
// http Handler Tests
//################################################################

func TestGetAllUsersHandler(t *testing.T) {
	// Define test cases
	tests := []struct {
		name       string
		method     string
		mockData   []interface{}
		mockError  error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "Incorrect Method",
			method:     http.MethodPost,
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:       "Database error",
			method:     http.MethodGet,
			mockData:   nil,
			mockError:  errors.New("mock error"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{ // This test should be last, because it saturates the cache
			name:   "Successful fetch",
			method: http.MethodGet,
			mockData: []interface{}{
				bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "jdoe", "Email": "john.doe@example.com", "Country": "USA",
					"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
				bson.M{"_id": "2", "first_name": "Jane", "last_name": "Smith", "nickname": "jsmith", "Email": "jane.smith@example.com", "Country": "UK",
					"password": "suP3rS3cret", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			},
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":"1","first_name":"John","last_name":"Doe","nickname":"jdoe","password":"moneyMoneyM0n3y","email":"john.doe@example.com","country":"USA","created_at":"2024-06-16T17:32:28.2136171Z","updated_at":"2024-06-16T17:32:28.2136171Z"},{"id":"2","first_name":"Jane","last_name":"Smith","nickname":"jsmith","password":"suP3rS3cret","email":"jane.smith@example.com","country":"UK","created_at":"2024-06-16T17:32:28.2136171Z","updated_at":"2024-06-16T17:32:28.2136171Z"}]`,
		},
		{
			name:       "Successful fetch Cache Hit",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":"1","first_name":"John","last_name":"Doe","nickname":"jdoe","password":"moneyMoneyM0n3y","email":"john.doe@example.com","country":"USA","created_at":"2024-06-16T17:32:28.2136171Z","updated_at":"2024-06-16T17:32:28.2136171Z"},{"id":"2","first_name":"Jane","last_name":"Smith","nickname":"jsmith","password":"suP3rS3cret","email":"jane.smith@example.com","country":"UK","created_at":"2024-06-16T17:32:28.2136171Z","updated_at":"2024-06-16T17:32:28.2136171Z"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				FindFunc: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return mocks.NewMockCursor(tt.mockData).Cursor, nil
				},
			})

			// Create a request to pass to the handler
			req, err := http.NewRequest(tt.method, "/userapi/getall", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler directly with the request and recorder
			getAllUsersHandler(rr, req)

			// Check the status code is what we expect
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", status, tt.wantStatus)
			}

			// Check the response body is what we expect
			if rr.Body.String() != tt.wantBody {
				t.Errorf("handler returned unexpected body: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestGetUsersHandler(t *testing.T) {
	// Define test cases
	tests := []struct {
		name            string
		method          string
		params          string
		mockData        []interface{}
		mockError       error
		expectedFilters bson.M
		expectedPage    int64
		expectedLimit   int64
		wantStatus      int
		wantBody        string
	}{
		{
			name:       "Incorrect Method",
			method:     http.MethodPost,
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:       "Database error",
			method:     http.MethodGet,
			mockData:   nil,
			mockError:  errors.New("mock error"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:   "Successful fetch",
			method: http.MethodGet,
			params: `?country=UK&nickname=j&createdAfter=2024-06-15T18%3A37%3A47.572Z&page=1&limit=50`,
			expectedFilters: bson.M{
				"country":    bson.M{`$options`: `i`, `$regex`: `UK`},
				"created_at": bson.M{`$gt`: time.Date(2024, time.June, 15, 18, 37, 47, 572000000, time.UTC)},
				"nickname":   bson.M{`$options`: `i`, `$regex`: `j`},
			},
			expectedPage:  1,
			expectedLimit: 50,
			mockData: []interface{}{
				bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "jdoe", "Email": "john.doe@example.com", "Country": "USA",
					"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			},
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":"1","first_name":"John","last_name":"Doe","nickname":"jdoe","password":"moneyMoneyM0n3y","email":"john.doe@example.com","country":"USA","created_at":"2024-06-16T17:32:28.2136171Z","updated_at":"2024-06-16T17:32:28.2136171Z"}]`,
		},
		{
			name:   "Successful fetch",
			method: http.MethodGet,
			params: `?country=Germany&nickname=jdoe&createdAfter=2024-02-15T18%3A37%3A47.572Z&page=1&limit=25`,
			expectedFilters: bson.M{
				"country":    bson.M{`$options`: `i`, `$regex`: `Germany`},
				"created_at": bson.M{`$gt`: time.Date(2024, time.February, 15, 18, 37, 47, 572000000, time.UTC)},
				"nickname":   bson.M{`$options`: `i`, `$regex`: `jdoe`},
			},
			expectedPage:  1,
			expectedLimit: 25,
			mockData: []interface{}{
				bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "jdoe", "Email": "john.doe@example.com", "Country": "USA",
					"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			},
			wantStatus: http.StatusOK,
			wantBody:   `[{"id":"1","first_name":"John","last_name":"Doe","nickname":"jdoe","password":"moneyMoneyM0n3y","email":"john.doe@example.com","country":"USA","created_at":"2024-06-16T17:32:28.2136171Z","updated_at":"2024-06-16T17:32:28.2136171Z"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				FindFunc: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}

					// Compare our filters to  ensure the request to mongo is correct
					if tt.expectedFilters != nil {
						if filter == nil {
							return nil, fmt.Errorf("expected filters: %v, got %v", tt.expectedFilters, filter)
						}

						bsonFilter, ok := filter.(bson.M)
						if !ok {
							return nil, fmt.Errorf("failed to cast filters to bson: %v", filter)
						}

						if !reflect.DeepEqual(bsonFilter, tt.expectedFilters) {
							return nil, fmt.Errorf("expected filters: %#v, got %#v", tt.expectedFilters, bsonFilter)
						}
					}

					if opts == nil {
						return nil, errors.New("no opts were provided")
					}

					if *opts[0].Skip != int64((tt.expectedPage-1)*tt.expectedLimit) {
						return nil, fmt.Errorf("expected skip %#v, got %#v", tt.expectedPage, *opts[0].Skip)
					}

					if *opts[0].Limit != tt.expectedLimit {
						return nil, fmt.Errorf("expected limit %#v, got %#v", tt.expectedLimit, *opts[0].Limit)
					}

					return mocks.NewMockCursor(tt.mockData).Cursor, nil
				},
			})

			// Create a request to pass to the handler
			req, err := http.NewRequest(tt.method, "/userapi/get"+tt.params, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler directly with the request and recorder
			getUsersHandler(rr, req)

			// Check the status code is what we expect
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", status, tt.wantStatus)
			}

			// Check the response body is what we expect
			if rr.Body.String() != tt.wantBody {
				t.Errorf("handler returned unexpected body: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestAddUserHandler(t *testing.T) {

	// Set out timenow function, to ensure our test is static
	timeNow = func() time.Time {
		return time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)
	}

	newUUID = func() string {
		return "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
	}

	// Define test cases
	tests := []struct {
		name            string
		method          string
		body            []byte
		mockData        interface{}
		mockError       error
		expectedFilters bson.M
		expectedUser    *data.User
		wantStatus      int
		wantBody        string
	}{
		{
			name:       "Incorrect Method",
			method:     http.MethodGet,
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:   "Database error",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com",
				"country": "UK"
			}`),
			mockError:  errors.New("mock error"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:       "Failed add, no first_name",
			method:     http.MethodPost,
			body:       []byte(`{}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed add, no last_name",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed add, no nickname",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed add, no valid pass",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "hello"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed add, no valid email",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "hello.hello"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed add, no country",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed add, username already exists",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com",
				"country": "UK"
			}`),
			expectedFilters: bson.M{"nickname": `Alchemist`},
			expectedUser:    &data.User{},
			mockData: bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "Alchemist", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Add user Successfully",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com",
				"country": "UK"
			}`),
			expectedFilters: bson.M{"nickname": `Alchemist`},
			expectedUser:    &data.User{ID: "8711e364-c83d-46fc-a3db-d6b2aee00d0f", FirstName: "Razzil", LastName: "Darkbrew", Nickname: "Alchemist", Password: "moneyMoneyM0n3y", Email: "Razzil.Darkbrew@example.com", Country: "UK", CreatedAt: time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC), UpdatedAt: time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)},
			mockData: bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "Dazzle", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			wantStatus: http.StatusOK,
			wantBody:   `{"id":"8711e364-c83d-46fc-a3db-d6b2aee00d0f","first_name":"Razzil","last_name":"Darkbrew","nickname":"Alchemist","password":"moneyMoneyM0n3y","email":"Razzil.Darkbrew@example.com","country":"UK","created_at":"2024-06-17T19:49:18.3688893Z","updated_at":"2024-06-17T19:49:18.3688893Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				FindOneFunc: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
					if tt.mockError != nil {
						return mongo.NewSingleResultFromDocument(bson.M{}, tt.mockError, nil)
					}

					// Compare our filters to  ensure the request to mongo is correct
					if tt.expectedFilters != nil {
						if filter == nil {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected filters: %v, got %v", tt.expectedFilters, filter), nil)
						}

						bsonFilter, ok := filter.(bson.M)
						if !ok {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("failed to cast filters to bson: %v", filter), nil)
						}

						if !reflect.DeepEqual(bsonFilter, tt.expectedFilters) {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected filters: %#v, got %#v", tt.expectedFilters, bsonFilter), nil)
						}
					}

					return mongo.NewSingleResultFromDocument(tt.mockData, nil, nil)
				},
				InsertOneFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
					if document == nil {
						return nil, fmt.Errorf("document must not be nil")
					}

					user, ok := document.(*data.User)
					if !ok {
						return nil, fmt.Errorf("user document does not match expected type")
					}

					if !reflect.DeepEqual(user, tt.expectedUser) {
						return nil, fmt.Errorf("expected user doest not match, want: %#v, got: %#v", tt.expectedUser, user)
					}

					return nil, nil
				},
			})

			// Create a request to pass to the handler
			req, err := http.NewRequest(tt.method, "/userapi/add", bytes.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler directly with the request and recorder
			addUserHandler(rr, req)

			// Check the status code is what we expect
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", status, tt.wantStatus)
			}

			// Check the response body is what we expect
			if rr.Body.String() != tt.wantBody {
				t.Errorf("handler returned unexpected body: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestUpdateUserHandler(t *testing.T) {

	// Set out timenow function, to ensure our test is static
	timeNow = func() time.Time {
		return time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)
	}

	newUUID = func() string {
		return "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
	}

	// Define test cases
	tests := []struct {
		name                  string
		method                string
		body                  []byte
		mockDataExisting      interface{}
		mockDataUpdated       interface{}
		mockError             error
		expectedFilters       bson.M
		expectedUserID        string
		expectedUpdateRequest bson.M
		wantStatus            int
		wantBody              string
	}{
		{
			name:       "Incorrect Method",
			method:     http.MethodGet,
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:   "Database error",
			method: http.MethodPost,
			body: []byte(`{
				"id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com",
				"country": "UK"
			}`),
			mockError:  errors.New("mock error"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:       "Failed update, no first_name",
			method:     http.MethodPost,
			body:       []byte(`{}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed update, no last_name",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed update, no nickname",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed update, no valid pass",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "hello"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed update, no valid email",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "hello.hello"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed update, no country",
			method: http.MethodPost,
			body: []byte(`{
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com"
			}`),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Failed update, new username already exists",
			method: http.MethodPost,
			body: []byte(`{
				"id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com",
				"country": "UK"
			}`),
			expectedFilters: bson.M{"nickname": `Alchemist`},
			mockDataExisting: bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "Alchemist", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "Updated User successfully",
			method: http.MethodPost,
			body: []byte(`{
				"id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Meepo",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com",
				"country": "UK"
			}`),
			expectedFilters:       bson.M{"nickname": `Meepo`},
			expectedUserID:        "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
			expectedUpdateRequest: bson.M{"$set": bson.M{"country": "UK", "email": "Razzil.Darkbrew@example.com", "first_name": "Razzil", "last_name": "Darkbrew", "nickname": "Meepo", "password": "moneyMoneyM0n3y", "updated_at": time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)}},
			mockDataExisting: bson.M{"_id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f", "first_name": "John", "last_name": "Doe", "nickname": "Dazzle", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			mockDataUpdated: bson.M{"_id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f", "first_name": "John", "last_name": "Doe", "nickname": "Meepo", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			wantStatus: http.StatusOK,
			wantBody:   `{"id":"8711e364-c83d-46fc-a3db-d6b2aee00d0f","first_name":"John","last_name":"Doe","nickname":"Meepo","password":"moneyMoneyM0n3y","email":"john.doe@example.com","country":"USA","created_at":"2024-06-16T17:32:28.2136171Z","updated_at":"2024-06-16T17:32:28.2136171Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				FindOneFunc: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
					if tt.mockError != nil {
						return mongo.NewSingleResultFromDocument(bson.M{}, tt.mockError, nil)
					}

					// Compare our filters to  ensure the request to mongo is correct
					if tt.expectedFilters != nil {
						if filter == nil {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected filters: %v, got %v", tt.expectedFilters, filter), nil)
						}

						bsonFilter, ok := filter.(bson.M)
						if !ok {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("failed to cast filters to bson: %v", filter), nil)
						}

						if !reflect.DeepEqual(bsonFilter, tt.expectedFilters) {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected filters: %#v, got %#v", tt.expectedFilters, bsonFilter), nil)
						}
					}

					return mongo.NewSingleResultFromDocument(tt.mockDataExisting, nil, nil)
				},
				FindOneAndUpdateFunc: func(ctx context.Context, filter interface{}, document interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
					if document == nil {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("document must not be nil"), nil)
					}

					bsonFilter, ok := filter.(bson.M)
					if !ok {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("no filters sent, expected filter on userid"), nil)
					}

					if bsonFilter["_id"] != tt.expectedUserID {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("id filter incorrect, does not match expected userid"), nil)
					}

					user, ok := document.(bson.M)
					if !ok {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("user document does not match expected type"), nil)
					}

					if !reflect.DeepEqual(user, tt.expectedUpdateRequest) {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected user doest not match, want: %#v, got: %#v", tt.expectedUpdateRequest, user), nil)
					}

					return mongo.NewSingleResultFromDocument(tt.mockDataUpdated, nil, nil)
				},
			})

			// Create a request to pass to the handler
			req, err := http.NewRequest(tt.method, "/userapi/update", bytes.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler directly with the request and recorder
			updateUserHandler(rr, req)

			// Check the status code is what we expect
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", status, tt.wantStatus)
			}

			// Check the response body is what we expect
			if rr.Body.String() != tt.wantBody {
				t.Errorf("handler returned unexpected body: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestDeleteUserHandler(t *testing.T) {

	// Define test cases
	tests := []struct {
		name            string
		method          string
		body            []byte
		expectedUserID  string
		mockDeleteCount int
		mockError       error
		wantStatus      int
		wantBody        string
	}{
		{
			name:       "Incorrect Method",
			method:     http.MethodGet,
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:   "Database error",
			method: http.MethodPost,
			body: []byte(`{
				"id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
			}`),
			mockError:  errors.New("mock error"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:   "Delete User successfully",
			method: http.MethodPost,
			body: []byte(`{
				"id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
			}`),
			expectedUserID:  "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
			mockDeleteCount: 1,
			wantStatus:      http.StatusOK,
		},
		{
			name:   "No users found to delete",
			method: http.MethodPost,
			body: []byte(`{
				"id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
			}`),
			expectedUserID:  "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
			mockDeleteCount: 0,
			wantStatus:      http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				DeleteOneFunc: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}

					bsonFilter, ok := filter.(bson.M)
					if !ok {
						return nil, fmt.Errorf("no filters sent, expected filter on userid")
					}

					if bsonFilter["_id"] != tt.expectedUserID {
						return nil, fmt.Errorf("id filter incorrect, does not match expected userid")
					}

					return &mongo.DeleteResult{
						DeletedCount: int64(tt.mockDeleteCount),
					}, nil
				},
			})

			// Create a request to pass to the handler
			req, err := http.NewRequest(tt.method, "/userapi/delete", bytes.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler directly with the request and recorder
			deleteUserHandler(rr, req)

			// Check the status code is what we expect
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", status, tt.wantStatus)
			}

			// Check the response body is what we expect
			if rr.Body.String() != tt.wantBody {
				t.Errorf("handler returned unexpected body: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestDeleteAllUserHandler(t *testing.T) {

	// Define test cases
	tests := []struct {
		name            string
		method          string
		body            []byte
		mockDeleteCount int
		mockError       error
		wantStatus      int
		wantBody        string
	}{
		{
			name:       "Incorrect Method",
			method:     http.MethodPost,
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:   "Database error",
			method: http.MethodGet,
			body: []byte(`{
				"id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Alchemist",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com",
				"country": "UK"
			}`),
			mockError:  errors.New("mock error"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "",
		},
		{
			name:   "Deleted all successfully",
			method: http.MethodGet,
			body: []byte(`{
				"id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
				"first_name": "Razzil",
				"last_name": "Darkbrew",
				"nickname": "Meepo",
				"password": "moneyMoneyM0n3y",
				"email": "Razzil.Darkbrew@example.com",
				"country": "UK"
			}`),
			mockDeleteCount: 1,
			wantStatus:      http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				DeleteManyFunc: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}

					return &mongo.DeleteResult{
						DeletedCount: int64(tt.mockDeleteCount),
					}, nil
				},
			})

			// Create a request to pass to the handler
			req, err := http.NewRequest(tt.method, "/userapi/deleteall", bytes.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler directly with the request and recorder
			deleteAllUsersHandler(rr, req)

			// Check the status code is what we expect
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", status, tt.wantStatus)
			}

			// Check the response body is what we expect
			if rr.Body.String() != tt.wantBody {
				t.Errorf("handler returned unexpected body: \n\rgot: \n\r%v \n\rwant: \n\r%v\n\r", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

//################################################################
// gRPC Handler Tests
//################################################################

func TestGetAllUsersGRPCHandler(t *testing.T) {

	// reset our cache
	db.UserStore.Clear()

	// Set out timenow function, to ensure our test is static
	timeNow = func() time.Time {
		return time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)
	}

	newUUID = func() string {
		return "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
	}

	// Define test cases
	tests := []struct {
		name          string
		mockData      []interface{}
		mockError     error
		expectedUsers []*pb.User
		expectedError bool
	}{
		{
			name:          "Database error",
			mockData:      nil,
			mockError:     errors.New("mock error"),
			expectedError: true,
		},
		{ // This test should be last, because it saturates the cache
			name: "Successful fetch",
			mockData: []interface{}{
				bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "jdoe", "Email": "john.doe@example.com", "Country": "USA",
					"password": "moneyMoneyM0n3y", "created_at": "2024-06-17T19:49:18.368889300Z", "updated_at": "2024-06-17T19:49:18.368889300Z"},
				bson.M{"_id": "2", "first_name": "Jane", "last_name": "Smith", "nickname": "jsmith", "Email": "jane.smith@example.com", "Country": "UK",
					"password": "suP3rS3cret", "created_at": "2024-06-17T19:49:18.368889300Z", "updated_at": "2024-06-17T19:49:18.368889300Z"},
			},
			expectedUsers: []*pb.User{
				{ID: "1", FirstName: "John", LastName: "Doe", Nickname: "jdoe", Password: "moneyMoneyM0n3y", Email: "john.doe@example.com", Country: "USA", CreatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)), UpdatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC))},
				{ID: "2", FirstName: "Jane", LastName: "Smith", Nickname: "jsmith", Password: "suP3rS3cret", Email: "jane.smith@example.com", Country: "UK", CreatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)), UpdatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC))},
			},
		},
		{
			name: "Successful fetch Cache Hit",
			expectedUsers: []*pb.User{
				{ID: "1", FirstName: "John", LastName: "Doe", Nickname: "jdoe", Password: "moneyMoneyM0n3y", Email: "john.doe@example.com", Country: "USA", CreatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)), UpdatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC))},
				{ID: "2", FirstName: "Jane", LastName: "Smith", Nickname: "jsmith", Password: "suP3rS3cret", Email: "jane.smith@example.com", Country: "UK", CreatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)), UpdatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC))},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				FindFunc: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return mocks.NewMockCursor(tt.mockData).Cursor, nil
				},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			response, err := grpcTestServer.GetAllUsers(ctx, nil)
			// If we got an error, but we didn't expect it. Then we error.
			// If we didn't get an error, but we expect one. Then we error.
			if (err != nil && !tt.expectedError) || (err == nil && tt.expectedError) {
				t.Errorf("handler returned an unexpected error: \n\rgot: \n\r%v", err)
			}

			// Exit early, because its an error scenario.
			if tt.expectedError {
				return
			}

			if len(response.Users) != len(tt.expectedUsers) {
				t.Errorf("handler returned unexpected response: \n\rgot: \n\r%#v \n\rwant: \n\r%#v\n\r", response.Users, tt.expectedUsers)
			}

			if !reflect.DeepEqual(response.Users, tt.expectedUsers) {
				for i := range response.Users {
					t.Errorf("handler returned unexpected response: \n\rgot: \n\r%#v \n\rwant: \n\r%#v\n\r", response.Users[i], tt.expectedUsers[i])
				}
			}
		})
	}
}

func TestGetUsersGRPCHandler(t *testing.T) {
	// Define test cases
	tests := []struct {
		name            string
		req             *pb.GetUsersRequest
		mockData        []interface{}
		mockError       error
		expectedFilters bson.M
		expectedError   bool
		expectedPage    int64
		expectedLimit   int64
		expectedUsers   []*pb.User
	}{
		{
			name:          "Database error",
			req:           &pb.GetUsersRequest{},
			mockData:      nil,
			mockError:     errors.New("mock error"),
			expectedError: true,
		},
		{
			name: "Successful fetch",
			req: &pb.GetUsersRequest{
				Country:      "UK",
				Nickname:     "j",
				CreatedAfter: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)),
				Page:         1,
				Limit:        50,
			},
			expectedFilters: bson.M{
				"country":    bson.M{`$options`: `i`, `$regex`: `UK`},
				"created_at": bson.M{`$gt`: time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)},
				"nickname":   bson.M{`$options`: `i`, `$regex`: `j`},
			},
			expectedPage:  1,
			expectedLimit: 50,
			mockData: []interface{}{
				bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "jdoe", "Email": "john.doe@example.com", "Country": "USA",
					"password": "moneyMoneyM0n3y", "created_at": "2024-06-17T19:49:18.368889300Z", "updated_at": "2024-06-17T19:49:18.368889300Z"},
			},
			expectedError: false,
			expectedUsers: []*pb.User{
				{ID: "1", FirstName: "John", LastName: "Doe", Nickname: "jdoe", Password: "moneyMoneyM0n3y", Email: "john.doe@example.com", Country: "USA", CreatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)), UpdatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC))},
			},
		},
		{
			name: "Successful fetch 2",
			req: &pb.GetUsersRequest{
				Country:      "Germany",
				Nickname:     "jdoe",
				CreatedAfter: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)),
				Page:         1,
				Limit:        25,
			},
			expectedFilters: bson.M{
				"country":    bson.M{`$options`: `i`, `$regex`: `Germany`},
				"created_at": bson.M{`$gt`: time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)},
				"nickname":   bson.M{`$options`: `i`, `$regex`: `jdoe`},
			},
			expectedPage:  1,
			expectedLimit: 25,
			expectedError: false,
			mockData: []interface{}{
				bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "jdoe", "Email": "john.doe@example.com", "Country": "USA",
					"password": "moneyMoneyM0n3y", "created_at": "2024-06-17T19:49:18.368889300Z", "updated_at": "2024-06-17T19:49:18.368889300Z"},
			},
			expectedUsers: []*pb.User{
				{ID: "1", FirstName: "John", LastName: "Doe", Nickname: "jdoe", Password: "moneyMoneyM0n3y", Email: "john.doe@example.com", Country: "USA", CreatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)), UpdatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC))},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				FindFunc: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}

					// Compare our filters to  ensure the request to mongo is correct
					if tt.expectedFilters != nil {
						if filter == nil {
							return nil, fmt.Errorf("expected filters: %v, got %v", tt.expectedFilters, filter)
						}

						bsonFilter, ok := filter.(bson.M)
						if !ok {
							return nil, fmt.Errorf("failed to cast filters to bson: %v", filter)
						}

						if !reflect.DeepEqual(bsonFilter, tt.expectedFilters) {
							return nil, fmt.Errorf("expected filters: %#v, got %#v", tt.expectedFilters, bsonFilter)
						}
					}

					if opts == nil {
						return nil, errors.New("no opts were provided")
					}

					if *opts[0].Skip != int64((tt.expectedPage-1)*tt.expectedLimit) {
						return nil, fmt.Errorf("expected skip %#v, got %#v", tt.expectedPage, *opts[0].Skip)
					}

					if *opts[0].Limit != tt.expectedLimit {
						return nil, fmt.Errorf("expected limit %#v, got %#v", tt.expectedLimit, *opts[0].Limit)
					}

					return mocks.NewMockCursor(tt.mockData).Cursor, nil
				},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			response, err := grpcTestServer.GetUsers(ctx, tt.req)
			// If we got an error, but we didn't expect it. Then we error.
			// If we didn't get an error, but we expect one. Then we error.
			if (err != nil && !tt.expectedError) || (err == nil && tt.expectedError) {
				t.Errorf("handler returned an unexpected error: \n\rgot: \n\r%v", err)
			}

			// Exit early, because its an error scenario.
			if tt.expectedError {
				return
			}

			if len(response.Users) != len(tt.expectedUsers) {
				t.Errorf("handler returned unexpected response: \n\rgot: \n\r%#v \n\rwant: \n\r%#v\n\r", response.Users, tt.expectedUsers)
			}

			if !reflect.DeepEqual(response.Users, tt.expectedUsers) {
				for i := range response.Users {
					t.Errorf("handler returned unexpected response: \n\rgot: \n\r%#v \n\rwant: \n\r%#v\n\r", response.Users[i], tt.expectedUsers[i])
				}
			}
		})
	}
}

func TestAddUserGRPCHandler(t *testing.T) {

	// Set out timenow function, to ensure our test is static
	timeNow = func() time.Time {
		return time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)
	}

	newUUID = func() string {
		return "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
	}

	// Define test cases
	tests := []struct {
		name             string
		req              *pb.AddUserRequest
		mockData         interface{}
		mockError        error
		expectedError    bool
		expectedFilters  bson.M
		expectedUser     *data.User
		expectedResponse *pb.User
	}{
		{
			name: "Database error",
			req: &pb.AddUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "Razzil.Darkbrew@example.com",
				Country:   "UK",
			},
			mockError:     errors.New("mock error"),
			expectedError: true,
		},
		{
			name:          "Failed add, no first_name",
			req:           &pb.AddUserRequest{},
			expectedError: true,
		},
		{
			name: "Failed add, no last_name",
			req: &pb.AddUserRequest{
				FirstName: "Razzil",
			},
			expectedError: true,
		},
		{
			name: "Failed add, no nickname",
			req: &pb.AddUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
			},
			expectedError: true,
		},
		{
			name: "Failed add, no valid pass",
			req: &pb.AddUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "testing",
			},
			expectedError: true,
		},
		{
			name: "Failed add, no valid email",
			req: &pb.AddUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "hello.hello",
				Country:   "UK",
			},
			expectedError: true,
		},
		{
			name: "Failed add, no country",
			req: &pb.AddUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "Razzil.Darkbrew@example.com",
			},
			expectedError: true,
		},
		{
			name: "Failed add, username already exists",
			req: &pb.AddUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "Razzil.Darkbrew@example.com",
				Country:   "UK",
			},
			expectedFilters: bson.M{"nickname": `Alchemist`},
			expectedUser:    &data.User{},
			mockData: bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "Alchemist", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			expectedError: true,
		},
		{
			name: "Add user Successfully",
			req: &pb.AddUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "Razzil.Darkbrew@example.com",
				Country:   "UK",
			},
			expectedFilters: bson.M{"nickname": `Alchemist`},
			expectedUser:    &data.User{ID: "8711e364-c83d-46fc-a3db-d6b2aee00d0f", FirstName: "Razzil", LastName: "Darkbrew", Nickname: "Alchemist", Password: "moneyMoneyM0n3y", Email: "Razzil.Darkbrew@example.com", Country: "UK", CreatedAt: time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC), UpdatedAt: time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)},
			mockData: bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "Dazzle", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			expectedResponse: &pb.User{ID: "8711e364-c83d-46fc-a3db-d6b2aee00d0f", FirstName: "Razzil", LastName: "Darkbrew", Nickname: "Alchemist", Password: "moneyMoneyM0n3y", Email: "Razzil.Darkbrew@example.com", Country: "UK", CreatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)), UpdatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC))},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				FindOneFunc: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
					if tt.mockError != nil {
						return mongo.NewSingleResultFromDocument(bson.M{}, tt.mockError, nil)
					}

					// Compare our filters to  ensure the request to mongo is correct
					if tt.expectedFilters != nil {
						if filter == nil {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected filters: %v, got %v", tt.expectedFilters, filter), nil)
						}

						bsonFilter, ok := filter.(bson.M)
						if !ok {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("failed to cast filters to bson: %v", filter), nil)
						}

						if !reflect.DeepEqual(bsonFilter, tt.expectedFilters) {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected filters: %#v, got %#v", tt.expectedFilters, bsonFilter), nil)
						}
					}

					return mongo.NewSingleResultFromDocument(tt.mockData, nil, nil)
				},
				InsertOneFunc: func(ctx context.Context, document interface{}) (*mongo.InsertOneResult, error) {
					if document == nil {
						return nil, fmt.Errorf("document must not be nil")
					}

					user, ok := document.(*data.User)
					if !ok {
						return nil, fmt.Errorf("user document does not match expected type")
					}

					if !reflect.DeepEqual(user, tt.expectedUser) {
						return nil, fmt.Errorf("expected user doest not match, want: %#v, got: %#v", tt.expectedUser, user)
					}

					return nil, nil
				},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			response, err := grpcTestServer.AddUser(ctx, tt.req)
			// If we got an error, but we didn't expect it. Then we error.
			// If we didn't get an error, but we expect one. Then we error.
			if (err != nil && !tt.expectedError) || (err == nil && tt.expectedError) {
				t.Errorf("handler returned an unexpected error: \n\rgot: \n\r%v", err)
			}

			// Exit early, because its an error scenario.
			if tt.expectedError {
				return
			}

			if !reflect.DeepEqual(response, tt.expectedResponse) {
				t.Errorf("handler returned unexpected response: \n\rgot: \n\r%#v \n\rwant: \n\r%#v\n\r", response, tt.expectedResponse)
			}
		})
	}
}

func TestUpdateUserGRPCHandler(t *testing.T) {

	// Set out timenow function, to ensure our test is static
	timeNow = func() time.Time {
		return time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)
	}

	newUUID = func() string {
		return "8711e364-c83d-46fc-a3db-d6b2aee00d0f"
	}

	// Define test cases
	tests := []struct {
		name                  string
		req                   *pb.UpdateUserRequest
		mockDataExisting      interface{}
		mockDataUpdated       interface{}
		mockError             error
		expectedError         bool
		expectedFilters       bson.M
		expectedUserID        string
		expectedUpdateRequest bson.M
		expectedUser          *data.User
		expectedResponse      *pb.User
	}{
		{
			name: "Database error",
			req: &pb.UpdateUserRequest{
				ID:        "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "Razzil.Darkbrew@example.com",
				Country:   "UK",
			},
			mockError:     errors.New("mock error"),
			expectedError: true,
		},
		{
			name:          "Failed update, no first_name",
			req:           &pb.UpdateUserRequest{},
			expectedError: true,
		},
		{
			name: "Failed update, no last_name",
			req: &pb.UpdateUserRequest{
				FirstName: "Razzil",
			},
			expectedError: true,
		},
		{
			name: "Failed update, no nickname",
			req: &pb.UpdateUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
			},
			expectedError: true,
		},
		{
			name: "Failed update, no valid pass",
			req: &pb.UpdateUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "testing",
			},
			expectedError: true,
		},
		{
			name: "Failed update, no valid email",
			req: &pb.UpdateUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "hello.hello",
				Country:   "UK",
			},
			expectedError: true,
		},
		{
			name: "Failed update, no country",
			req: &pb.UpdateUserRequest{
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "Razzil.Darkbrew@example.com",
			},
			expectedError: true,
		},
		{
			name: "Failed update, username already exists",
			req: &pb.UpdateUserRequest{
				ID:        "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "Razzil.Darkbrew@example.com",
				Country:   "UK",
			},
			expectedFilters: bson.M{"nickname": `Alchemist`},
			expectedUser:    &data.User{},
			mockDataExisting: bson.M{"_id": "1", "first_name": "John", "last_name": "Doe", "nickname": "Alchemist", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			expectedError: true,
		},
		{
			name: "Updated User successfully",
			req: &pb.UpdateUserRequest{
				ID:        "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
				FirstName: "Razzil",
				LastName:  "Darkbrew",
				Nickname:  "Alchemist",
				Password:  "moneyMoneyM0n3y",
				Email:     "Razzil.Darkbrew@example.com",
				Country:   "UK",
			},
			expectedFilters:       bson.M{"nickname": `Alchemist`},
			expectedUserID:        "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
			expectedUpdateRequest: bson.M{"$set": bson.M{"country": "UK", "email": "Razzil.Darkbrew@example.com", "first_name": "Razzil", "last_name": "Darkbrew", "nickname": "Alchemist", "password": "moneyMoneyM0n3y", "updated_at": time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)}},
			expectedUser:          &data.User{ID: "8711e364-c83d-46fc-a3db-d6b2aee00d0f", FirstName: "Razzil", LastName: "Darkbrew", Nickname: "Alchemist", Password: "moneyMoneyM0n3y", Email: "Razzil.Darkbrew@example.com", Country: "UK", CreatedAt: time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC), UpdatedAt: time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)},
			mockDataExisting: bson.M{"_id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f", "first_name": "John", "last_name": "Doe", "nickname": "Dazzle", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-16T17:32:28.2136171Z", "updated_at": "2024-06-16T17:32:28.2136171Z"},
			mockDataUpdated: bson.M{"_id": "8711e364-c83d-46fc-a3db-d6b2aee00d0f", "first_name": "John", "last_name": "Doe", "nickname": "Meepo", "Email": "john.doe@example.com", "Country": "USA",
				"password": "moneyMoneyM0n3y", "created_at": "2024-06-17T19:49:18.368889300Z", "updated_at": "2024-06-17T19:49:18.368889300Z"},
			expectedResponse: &pb.User{ID: "8711e364-c83d-46fc-a3db-d6b2aee00d0f", FirstName: "John", LastName: "Doe", Nickname: "Meepo", Password: "moneyMoneyM0n3y", Email: "john.doe@example.com", Country: "USA", CreatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC)), UpdatedAt: timestamppb.New(time.Date(2024, time.June, 17, 19, 49, 18, 368889300, time.UTC))},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				FindOneFunc: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
					if tt.mockError != nil {
						return mongo.NewSingleResultFromDocument(bson.M{}, tt.mockError, nil)
					}

					// Compare our filters to  ensure the request to mongo is correct
					if tt.expectedFilters != nil {
						if filter == nil {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected filters: %v, got %v", tt.expectedFilters, filter), nil)
						}

						bsonFilter, ok := filter.(bson.M)
						if !ok {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("failed to cast filters to bson: %v", filter), nil)
						}

						if !reflect.DeepEqual(bsonFilter, tt.expectedFilters) {
							return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected filters: %#v, got %#v", tt.expectedFilters, bsonFilter), nil)
						}
					}

					return mongo.NewSingleResultFromDocument(tt.mockDataExisting, nil, nil)
				},
				FindOneAndUpdateFunc: func(ctx context.Context, filter interface{}, document interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
					if document == nil {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("document must not be nil"), nil)
					}

					bsonFilter, ok := filter.(bson.M)
					if !ok {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("no filters sent, expected filter on userid"), nil)
					}

					if bsonFilter["_id"] != tt.expectedUserID {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("id filter incorrect, does not match expected userid"), nil)
					}

					user, ok := document.(bson.M)
					if !ok {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("user document does not match expected type"), nil)
					}

					if !reflect.DeepEqual(user, tt.expectedUpdateRequest) {
						return mongo.NewSingleResultFromDocument(bson.M{}, fmt.Errorf("expected user doest not match, want: %#v, got: %#v", tt.expectedUpdateRequest, user), nil)
					}

					return mongo.NewSingleResultFromDocument(tt.mockDataUpdated, nil, nil)
				},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			response, err := grpcTestServer.UpdateUser(ctx, tt.req)
			// If we got an error, but we didn't expect it. Then we error.
			// If we didn't get an error, but we expect one. Then we error.
			if (err != nil && !tt.expectedError) || (err == nil && tt.expectedError) {
				t.Errorf("handler returned an unexpected error: \n\rgot: \n\r%v", err)
			}

			// Exit early, because its an error scenario.
			if tt.expectedError {
				return
			}

			if !reflect.DeepEqual(response, tt.expectedResponse) {
				t.Errorf("handler returned unexpected response: \n\rgot: \n\r%#v \n\rwant: \n\r%#v\n\r", response, tt.expectedResponse)
			}
		})
	}
}

func TestDeleteUserGRPCHandler(t *testing.T) {

	// Define test cases
	tests := []struct {
		name            string
		req             *pb.DeleteUserRequest
		expectedUserID  string
		expectedError   bool
		mockDeleteCount int
		mockError       error
		wantStatus      int
		wantBody        string
	}{
		{
			name: "Database error",
			req: &pb.DeleteUserRequest{
				ID: "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
			},
			mockError:     errors.New("mock error"),
			expectedError: true,
			wantBody:      "",
		},
		{
			name: "Delete User successfully",
			req: &pb.DeleteUserRequest{
				ID: "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
			},
			expectedUserID:  "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
			mockDeleteCount: 1,
			expectedError:   false,
			wantStatus:      http.StatusOK,
		},
		{
			name: "No users found to delete",
			req: &pb.DeleteUserRequest{
				ID: "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
			},
			expectedUserID:  "8711e364-c83d-46fc-a3db-d6b2aee00d0f",
			mockDeleteCount: 0,
			expectedError:   true,
			wantStatus:      http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the userCollection.Find method
			db.SetCollection(&mocks.MongoCollection{
				DeleteOneFunc: func(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}

					bsonFilter, ok := filter.(bson.M)
					if !ok {
						return nil, fmt.Errorf("no filters sent, expected filter on userid")
					}

					if bsonFilter["_id"] != tt.expectedUserID {
						return nil, fmt.Errorf("id filter incorrect, does not match expected userid")
					}

					return &mongo.DeleteResult{
						DeletedCount: int64(tt.mockDeleteCount),
					}, nil
				},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err := grpcTestServer.DeleteUser(ctx, tt.req)
			// If we got an error, but we didn't expect it. Then we error.
			// If we didn't get an error, but we expect one. Then we error.
			if (err != nil && !tt.expectedError) || (err == nil && tt.expectedError) {
				t.Errorf("handler returned an unexpected error: \n\rgot: \n\r%v", err)
			}

			// Exit early, because its an error scenario.
			if tt.expectedError {
				return
			}
		})
	}
}
