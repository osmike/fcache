package core

import (
	"container/list"
	"sync"
	"time"
)

// Storage is a generic, thread-safe LRU cache for values of type Val.
//
// It supports per-entry TTL expiration, capacity-based eviction, and LRU ordering.
// Each entry is moved to the front of the usage list on access.
type Storage[Val any] struct {
	mu       sync.RWMutex
	data     map[string]*StorageItem[Val] // map key to cached value
	ll       *list.List                   // list of keys, front is most recently used
	elems    map[string]*list.Element     // map key to list element
	capacity int
	ttl      time.Duration // time-to-live for cache entries

	cleanInterval  time.Duration // interval for periodic cleanup of expired entries
	stopCleanup    chan struct{} // channel to signal cleanup goroutine to stop
	cleanupRunning bool          // indicates if cleanup goroutine is active
}

// StorageItem represents a single cache entry, holding the stored value
// and its insertion timestamp for TTL validation.
type StorageItem[V any] struct {
	Value     V         // cached value
	Timestamp time.Time // timestamp of last insert
}

// StorageStat holds statistics and a snapshot of cache items.
// Entries are listed in LRU order, from most to least recent.
type StorageStat[V any] struct {
	Entries int              // number of entries in cache
	Items   []StorageItem[V] // items in LRU order, from most to least recent
}

// NewStorage initializes a new Storage with specified TTL and capacity.
//
//   - ttl: Time-to-live for each cache entry.
//   - capacity: Maximum number of cache entries (default: 1000 if <= 0).
//   - cleanInterval: Interval for periodic cleanup of expired entries.
//
// Returns a pointer to the initialized Storage.
func NewStorage[V any](ttl time.Duration, capacity int, cleanInterval time.Duration) *Storage[V] {
	if capacity <= 0 {
		capacity = 1000
	}
	s := &Storage[V]{
		data:           make(map[string]*StorageItem[V]),
		ll:             list.New(),
		elems:          make(map[string]*list.Element),
		capacity:       capacity,
		ttl:            ttl,
		cleanInterval:  cleanInterval,
		stopCleanup:    make(chan struct{}),
		cleanupRunning: false,
	}

	return s
}

// Get retrieves the cached value for the given key.
//
// If the entry exists and is not expired, it moves the entry to the front of the LRU list.
// Returns (value, true) if found and valid; otherwise returns (zero, false).
func (s *Storage[V]) Get(key string) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if elem, ok := s.elems[key]; ok {
		s.ll.MoveToFront(elem)
		val := s.data[key]
		// Check if the item is still valid based on TTL
		if time.Since(val.Timestamp) > s.ttl {
			s.deleteProxy(key)
			var zero V
			return zero, false
		}
		return val.Value, true
	}
	var zero V
	return zero, false
}

// Set inserts or updates the cache entry for the given key with the provided value.
//
// It timestamps the entry and moves it to the front of the LRU list.
// If capacity is exceeded, the least recently used entry is evicted.
// Starts the cleanup goroutine if not already running.
func (s *Storage[V]) Set(key string, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := &StorageItem[V]{
		Value:     value,
		Timestamp: time.Now(),
	}
	// insert new entry
	elem := s.ll.PushFront(key)
	s.elems[key] = elem
	s.data[key] = item

	// evict least recently used if over capacity
	if len(s.data) > s.capacity {
		tail := s.ll.Back()
		if tail != nil {
			oldKey := tail.Value.(string)
			s.ll.Remove(tail)
			delete(s.elems, oldKey)
			delete(s.data, oldKey)
		}
	}
	// If cleanup is not running, start it
	if !s.cleanupRunning {
		s.cleanupRunning = true
		go s.startCleanup(s.cleanInterval) // start cleanup every 5 minutes
	}
}

// Delete removes the cache entry for the given key, if present,
// updating both the map and the LRU list.
func (s *Storage[V]) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteProxy(key)
}

// deleteProxy is an internal helper to remove a key from the cache and LRU list.
// If the cache becomes empty, it stops the cleanup goroutine.
func (s *Storage[V]) deleteProxy(key string) {
	if elem, ok := s.elems[key]; ok {
		s.ll.Remove(elem)
		delete(s.elems, key)
		delete(s.data, key)
		if len(s.data) == 0 && s.cleanupRunning {
			// If no entries left, stop the cleanup goroutine
			s.cleanupRunning = false
			close(s.stopCleanup) // signal cleanup goroutine to stop
		}
	}
}

// startCleanup launches a ticker that triggers cleanupExpired at the given interval.
// The cleanup goroutine stops when the cache becomes empty.
func (s *Storage[V]) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.cleanupExpired() // perform cleanup
		case <-s.stopCleanup:
			return
		}
	}
}

// cleanupExpired removes all entries whose TTL has elapsed.
func (s *Storage[V]) cleanupExpired() {
	now := time.Now()
	s.mu.Lock()
	// collect keys to delete to avoid mutation during iteration
	var expired []string
	for key, item := range s.data {
		if now.Sub(item.Timestamp) > s.ttl {
			expired = append(expired, key)
		}
	}
	// delete expired entries
	for _, key := range expired {
		s.deleteProxy(key)
	}
	s.mu.Unlock()
}
