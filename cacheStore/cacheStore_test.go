package cacheStore

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// Test creating a new store and retrieving data from it.
func TestNewStoreAndRetrieveData(t *testing.T) {
	store := NewStore[string, string]("exampleStore", 1*time.Second)

	// Attempt to get data that's not yet cached.
	val, err := store.GetData("key1", fetchMockData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "data for key1" {
		t.Fatalf("expected 'data for key1', got %v", val)
	}

	// Attempt to get data that should now be cached.
	val, err = store.GetData("key1", fetchMockData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "data for key1" {
		t.Fatalf("expected 'data for key1', got %v", val)
	}
}

// Test the cache expiration and automatic cleanup.
func TestCacheExpiryAndCleanup(t *testing.T) {
	store := NewStore[string, string]("exampleStore", 1*time.Second)

	// Insert data into the cache.
	_, err := store.GetData("key1", fetchMockData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for the cache to expire.
	time.Sleep(2 * time.Second)

	// Attempt to get data again, which should reload it.
	val, err := store.GetData("key1", fetchAlternateMockData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "alt data for key1" {
		t.Fatalf("expected 'alt data for key1', got %v", val)
	}
}

// Test error handling when the data function fails.
func TestFetchDataWithError(t *testing.T) {
	store := NewStore[string, string]("exampleStore", 1*time.Second)

	// Attempt to fetch data that will cause an error.
	_, err := store.GetData("error", fetchMockData)
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}
	if err.Error() != "mock error" {
		t.Fatalf("expected 'mock error', got %v", err)
	}
}

// Test concurrent access to the cache store.
func TestConcurrentDataAccess(t *testing.T) {
	store := NewStore[string, string]("exampleStore", 1*time.Second)

	var wg sync.WaitGroup
	keys := []string{"key1", "key2", "key3", "key4", "key5"}

	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			_, err := store.GetData(k, fetchMockData)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(key)
	}

	wg.Wait()
}

// Mock data function for testing.
func fetchMockData(key string) (string, error) {
	if key == "error" {
		return "", errors.New("mock error")
	}
	return "data for " + key, nil
}

// Alternate mock data function for testing.
func fetchAlternateMockData(key string) (string, error) {
	if key == "error" {
		return "", errors.New("mock error")
	}
	return "alt data for " + key, nil
}
