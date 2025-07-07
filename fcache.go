// Package fcache provides a generic, concurrent-safe caching layer for expensive or long-running functions.
//
// # Overview
//
// fcache enables memoization, in-flight request deduplication, time-based expiration, and LRU-based capacity limiting for any Go function.
// It is designed for production use, with a focus on correctness, performance, and code clarity.
//
// ## Features
//
//   - Memoization: Avoids redundant computations by caching results for identical input parameters.
//   - In-flight Request Deduplication: Ensures only one execution for concurrent calls with the same input; others wait for the result.
//   - Expiration: Each cache entry expires after a configurable TTL (default: 5 minutes).
//   - Capacity Limit: The cache holds up to a configurable number of entries (default: 1000), evicting the least recently used (LRU) entries when full.
//   - Concurrency Safety: All operations are safe for concurrent use.
//   - Extensibility: Optional hooks for instrumentation and custom logic.
//
// ## Usage Example
//
//	// A long-running function
//	func fetchDataFromRemote(timeMS int) (string, error) { ... }
//
//	// Wrap with caching
//	cachedFetch := fcache.NewCachedFunction(fetchDataFromRemote, nil, nil)
//	result, err := cachedFetch(2000)
//
// ## Customization
//
//   - Use the Config struct to customize TTL and capacity.
//   - Use the Hooks struct to add custom logic (e.g., logging, metrics).
//
// See package documentation and tests for more details.
package fcache

import (
	"github.com/osmike/fcache/internal/core"
	"github.com/osmike/fcache/internal/lib/hooks"
)

// CachedFunc is a generic function type that can be wrapped with caching.
// K is the input parameter type, V is the result type.
type CachedFunc[K any, V any] = core.CachedFunc[K, V]

// Config defines cache configuration options such as TTL and capacity.
type Config = core.Config

// Hooks provides optional hooks for cache events (e.g., on hit, miss, eviction).
type Hooks = hooks.Hooks

// NewCachedFunction wraps a function with a concurrent-safe caching layer.
//
//   - fn: The function to cache. Must be of type func(K) (V, error).
//   - opts: Optional cache configuration (TTL, capacity). Pass nil for defaults.
//   - hooks: Optional hooks for cache events. Pass nil if not needed.
//
// Returns a function with the same signature as fn, but with caching applied.
//
// Example:
//
//	cachedFetch := fcache.NewCachedFunction(fetchDataFromRemote, nil, nil)
//
// See package documentation for details.
func NewCachedFunction[K any, V any](fn CachedFunc[K, V], opts *Config, hooks *hooks.Hooks) CachedFunc[K, V] {
	return core.NewCachedFunction(fn, opts, hooks)
}
