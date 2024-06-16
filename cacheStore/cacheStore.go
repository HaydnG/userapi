// Taken from https://github.com/HaydnG/carHiringWebsite/blob/master/cacheStore/cacheStore.go
// And updated to use generics
// TODO: Implement a soft expiry method,
//	this will reduct cache misses, via serving up the previous cache while fetching in the bachground

package cacheStore

import (
	"sync"
	"time"
)

// NewStore creates a new generic store with a given name and duration.
// The duration controls how long the data will be cached
func NewStore[K comparable, V any](name string, duration time.Duration) *store[K, V] {
	s := &store[K, V]{
		name:            name,
		data:            make(map[K]cacheItem[K, V]),
		duration:        duration,
		cleanUpInterval: 1,
		cleanUpActive:   true,
	}
	s.cleanUpJob()
	return s
}

// cacheItem represents a cached item with generic type.
type cacheItem[K comparable, V any] struct {
	key     K
	created time.Time
	data    V
}

// store is a generic cache store.
type store[K comparable, V any] struct {
	lock            sync.RWMutex
	name            string
	data            map[K]cacheItem[K, V]
	duration        time.Duration
	cleanUpInterval int
	cleanUpActive   bool
}

// cleanUpJob periodically cleans up expired cache items.
func (s *store[K, V]) cleanUpJob() {
	duration := s.duration
	sleepInterval := time.Duration(s.cleanUpInterval) * time.Second

	go func() {
		for s.cleanUpActive {
			time.Sleep(sleepInterval)
			if !s.cleanUpActive {
				return
			}
			if len(s.data) < 1 {
				continue
			}
			s.lock.Lock()
			now := time.Now()

			for key, item := range s.data {
				if now.Sub(item.created) >= duration {
					delete(s.data, key)
				}
			}
			s.lock.Unlock()
		}
	}()
}

// GetData retrieves data from the cache or loads it using dataFunction if not present.
func (s *store[K, V]) GetData(key K, dataFunction func(key K) (V, error)) (V, error) {
	var err error
	var zeroValue V

	s.lock.RLock()
	item, ok := s.data[key]
	if !ok {
		s.lock.RUnlock()
		s.lock.Lock()
		defer s.lock.Unlock()
		item, ok = s.data[key]
		if !ok {
			item, err = s.addData(key, dataFunction)
			if err != nil {
				return zeroValue, err
			}
		} else {
			s.lock.RUnlock()
		}
	} else {
		s.lock.RUnlock()
	}

	if ok && time.Since(item.created) >= (s.duration*time.Second) {
		s.lock.Lock()
		defer s.lock.Unlock()
		item, err = s.addData(key, dataFunction)
		if err != nil {
			return zeroValue, err
		}
	}

	return item.data, nil
}

// addData adds new data to the cache by invoking dataFunction.
func (s *store[K, V]) addData(key K, dataFunction func(key K) (V, error)) (cacheItem[K, V], error) {
	data, err := dataFunction(key)
	if err != nil {
		return cacheItem[K, V]{}, err
	}

	item := cacheItem[K, V]{
		key:     key,
		created: time.Now(),
		data:    data,
	}
	s.data[key] = item
	return item, nil
}
