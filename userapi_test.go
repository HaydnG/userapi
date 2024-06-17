package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"userapi/db"
	"userapi/mocks"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PrettyPrintJSON formats JSON with indentation.
func PrettyPrintJSON(v interface{}) string {
	prettyJSON, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err.Error()
	}
	return string(prettyJSON)
}

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
		{ // This test should be last, because it saturates the cache
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
		{ // This test should be last, because it saturates the cache
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
		{ // This test should be last, because it saturates the cache
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

					if tt.expectedFilters != nil {
						if filter == nil {
							if tt.mockError != nil {
								return nil, tt.mockError
							}
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
