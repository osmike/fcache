// Package core implements the core logic for concurrent, memoizing, and capacity-limited function caching.
//
// This package provides the internal implementation for the fcache package, enabling production-grade caching for expensive or long-running functions.
//
// # Features
//
//   - Memoization: Caches results for identical input parameters to avoid redundant computation.
//   - In-flight Request Deduplication: Ensures only one execution for concurrent calls with the same input; others wait for the result.
//   - Expiration: Each cache entry expires after a configurable TTL (default: 5 minutes).
//   - Capacity Limit: The cache holds up to a configurable number of entries (default: 1000), evicting the least recently used (LRU) entries when full.
//   - Concurrency Safety: All operations are safe for concurrent use.
//   - Extensibility: Optional hooks for instrumentation and custom logic.
//
// # Usage
//
// This package is not intended for direct use. Use the fcache package for a public API.
//
// # Type Parameters
//
//   - K: The type of the function argument (must be serializable to a cache key).
//   - V: The type of the function result.
//
// # Example
//
//	// Wrap a function with caching:
//	cached := core.NewCachedFunction(fetchData, &core.Config{TTL: 2*time.Minute, Capacity: 500}, nil)
//	result, err := cached(123)
package core

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/osmike/fcache/internal/lib/errs"
	"github.com/osmike/fcache/internal/lib/hooks"
	"github.com/osmike/fcache/internal/lib/keygen"
)

// Default settings for cache TTL and maximum size.
const (
	defaultTTL             = 5 * time.Minute
	defaultMaxSize         = 1000
	defaultCleanupInterval = 1 * time.Minute // Default interval for periodic cleanup
)

// ErrPanic is returned if a panic occurs in the cached function.
var ErrPanic = errors.New("panic occurred in cached function")

// CachedFunc wraps a user-provided function with caching behavior.
//
// K is the input parameter type (must be serializable to a cache key).
// V is the return type.
//
// The function must have the signature: func(arg K) (V, error)
type CachedFunc[K any, V any] func(arg K) (V, error)

// Config configures the cache behavior.
//
//   - TTL: Time-to-live for each cache entry (default: 5 minutes).
//   - Capacity: Maximum number of cache entries (default: 1000).
//   - CleanupInterval: Interval for periodic cleanup of expired entries (default: 1 minute).
type Config struct {
	TTL             time.Duration // Time-to-live for each cache entry.
	Capacity        int           // Maximum number of cache entries.
	CleanupInterval time.Duration // Interval for periodic cleanup (if implemented).
}

// inflightCall deduplicates concurrent calls for the same key.
// It holds the result and error, and a wait group for synchronization.
type inflightCall[V any] struct {
	wg  sync.WaitGroup // Waits for the function execution to complete
	val V              // Result value
	err error          // Result error
}

// cache is the internal structure that manages the cache state and logic.
//
// It holds the user function, cache storage, in-flight deduplication map, configuration, and hooks.
type cache[K any, V any] struct {
	mu       sync.Mutex                  // Protects inflight and cache state
	fn       CachedFunc[K, V]            // User-provided function to cache
	store    *Storage[V]                 // Underlying storage for cached values
	inflight map[string]*inflightCall[V] // Tracks in-flight requests for deduplication
	cfg      *Config                     // Cache configuration
	hooks    *hooks.Hooks                // Hooks for lifecycle events
}

// NewCachedFunction returns a CachedFunc that wraps fn with caching logic.
//
// The returned function provides memoization, in-flight deduplication, TTL, and LRU eviction.
// You can pass optional TTL and max-size options via Config.
//
//   - fn: The function to cache. Must be of type func(K) (V, error).
//   - opts: Optional cache configuration (TTL, capacity, cleanup interval). Pass nil for defaults.
//   - h: Optional hooks for cache events. Pass nil if not needed.
//
// Returns a function with the same signature as fn, but with caching applied.
func NewCachedFunction[K any, V any](fn CachedFunc[K, V], opts *Config, h *hooks.Hooks) CachedFunc[K, V] {

	// Default config if nil
	if opts == nil {
		opts = &Config{}
	}
	// Apply defaults
	if opts.TTL <= 0 {
		opts.TTL = defaultTTL
	}
	if opts.Capacity <= 0 {
		opts.Capacity = defaultMaxSize
	}
	if opts.CleanupInterval <= 0 {
		opts.CleanupInterval = defaultCleanupInterval
	}
	// Default hooks if nil
	if h == nil {
		h = &hooks.Hooks{}
	}

	c := &cache[K, V]{
		fn:       fn,
		store:    NewStorage[V](opts.TTL, opts.Capacity, opts.CleanupInterval),
		inflight: make(map[string]*inflightCall[V]),
		cfg:      opts,
		hooks:    h,
	}

	return c.call
}

// call executes the cached function with deduplication, TTL, and LRU eviction.
//
// It ensures only one execution per unique key is in-flight at a time.
// If a panic occurs in the user function, it is caught and returned as an error.
//
//   - arg: The input parameter for the cached function.
//   - Returns: The result value and error from the function or cache.
func (c *cache[K, V]) call(arg K) (val V, err error) {
	var zero V
	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			switch x := r.(type) {
			case error:
				panicErr = errs.NewError(ErrPanic, map[string]interface{}{
					"panic": x,
				})
			case string:
				panicErr = errs.NewError(ErrPanic, map[string]interface{}{
					"panic": x,
				})
			default:
				panicErr = errs.NewError(ErrPanic, map[string]interface{}{
					"panic": fmt.Errorf("%v", x),
				})
			}
			// Safely log the panic error if a logging hook is defined.
			if c.hooks.LogError != nil {
				defer func() { recover() }()
				c.hooks.LogError(panicErr)
			}
			err = panicErr
			val = zero // Reset value to zero value of type V
		}
	}()
	key, err := keygen.BuildKey(arg)
	if err != nil {
		return zero, err
	}

	// Fast path: check if value is already cached.
	if val, found := c.store.Get(key); found {
		// Run the OnGet hook if defined.
		if c.hooks.OnGet != nil {
			c.hooks.Run(c.hooks.OnGet, arg)
		}
		return val, nil
	}

	c.mu.Lock()
	// Check if another goroutine is already computing this key.
	if ic, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		ic.wg.Wait()
		return ic.val, ic.err
	}

	// Mark this key as in-flight.
	ic := &inflightCall[V]{}
	ic.wg.Add(1)
	c.inflight[key] = ic
	c.mu.Unlock()

	// Run the OnExecute hook if defined.
	if c.hooks.OnExecute != nil {
		c.hooks.Run(c.hooks.OnExecute, arg)
	}
	// Call the underlying function outside the lock.
	val, err = c.fn(arg)
	// Run the OnDone hook if defined.
	if c.hooks.OnDone != nil {
		c.hooks.Run(c.hooks.OnDone, arg)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Remove in-flight marker.
	delete(c.inflight, key)
	// Notify waiters with result.
	ic.val = val
	ic.err = err
	ic.wg.Done()

	if err != nil {
		// If the function returned an error, we do not cache it.
		// Log the error if a logging hook is defined.
		if c.hooks.LogError != nil {
			c.hooks.LogError(err)
		}
		return zero, err
	}

	// Store successful result in cache.
	c.store.Set(key, val)
	if c.hooks.OnSet != nil {
		c.hooks.Run(c.hooks.OnSet, arg)
	}
	return val, nil
}
