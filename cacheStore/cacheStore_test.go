package cacheStore

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// Mock data function for testing.
func mockDataFunction(key string) (string, error) {
	if key == "error" {
		return "", errors.New("mock error")
	}
	return "value for " + key, nil
}

// Mock data function for testing.
func mockDataFunctionTwo(key string) (string, error) {
	if key == "error" {
		return "", errors.New("mock error")
	}
	return "two: value for " + key, nil
}

// Test creating a new store and retrieving data.
func TestNewStoreAndGetData(t *testing.T) {
	store := NewStore[string, string]("testStore", 1*time.Second)

	// Retrieve data that is not in the cache.
	value, err := store.GetData("key1", mockDataFunction)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "value for key1" {
		t.Fatalf("expected 'value for key1', got %v", value)
	}

	// Retrieve data that is already in the cache.
	value, err = store.GetData("key1", mockDataFunction)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "value for key1" {
		t.Fatalf("expected 'value for key1', got %v", value)
	}
}

// Test cache expiration and cleanup.
func TestCacheExpirationAndCleanup(t *testing.T) {
	store := NewStore[string, string]("testStore", 1*time.Second)

	// Add data to the cache.
	_, err := store.GetData("key1", mockDataFunction)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for the cache to expire.
	time.Sleep(2 * time.Second)

	// Try to retrieve data, which should trigger a cache miss and reload.
	value, err := store.GetData("key1", mockDataFunctionTwo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "two: value for key1" {
		t.Fatalf("expected 'value for key1', got %v", value)
	}
}

// Test handling of data function errors.
func TestDataFunctionError(t *testing.T) {
	store := NewStore[string, string]("testStore", 1*time.Second)

	// Try to retrieve data using a key that causes the data function to return an error.
	_, err := store.GetData("error", mockDataFunction)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "mock error" {
		t.Fatalf("expected 'mock error', got %v", err)
	}
}

// Test concurrent access to the store.
func TestConcurrentAccess(t *testing.T) {
	store := NewStore[string, string]("testStore", 1*time.Second)

	var wg sync.WaitGroup
	keys := []string{"key1", "key2", "key3", "key4", "key5"}

	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			_, err := store.GetData(k, mockDataFunction)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(key)
	}

	wg.Wait()
}
