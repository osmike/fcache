# fcache

A high-performance, production-grade caching library for Go functions. 

---

## 🚀 Overview

**fcache** provides a generic, concurrent-safe caching layer for expensive or long-running Go functions. It is designed for production environments, focusing on correctness, performance, and code clarity. With fcache, you can easily add memoization, in-flight request deduplication, time-based expiration, and LRU-based capacity limiting to any function in your codebase.

### Key Features
- **Memoization**: Avoid redundant computations by caching results for identical input parameters.
- **In-flight Request Deduplication**: Ensures only one execution for concurrent calls with the same input; others wait for the result.
- **Expiration**: Each cache entry expires after a configurable TTL (default: 5 minutes).
- **Capacity Limit**: The cache holds up to a configurable number of entries (default: 1000), evicting the least recently used (LRU) entries when full.
- **Concurrency Safety**: All operations are safe for concurrent use.
- **Extensibility**: Optional hooks for instrumentation and custom logic.

---

## 📦 Installation

```sh
go get github.com/osmike/fcache
```

---

## 🛠️ API Usage

### Basic Example

```go
package main

import (
	"fmt"
	"time"
	"github.com/osmike/fcache"
)

func main() {
	cachedFunction := fcache.NewCachedFunction(heavyComputation, nil, nil)
	fmt.Printf("[%v] Starting heavy computation...\n", time.Now().Truncate(time.Second))
	res, err := cachedFunction(2000 * time.Millisecond)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("[%v] Heavy computation completed, result - %s.\n", time.Now().Truncate(time.Second), res)

	fmt.Printf("[%v] Starting cached heavy computation...\n", time.Now().Truncate(time.Second))
	_, err = cachedFunction(2000 * time.Millisecond)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("[%v] Heavy computation completed, result cached - %s.\n", time.Now().Truncate(time.Second), res)
}

func heavyComputation(t time.Duration) (string, error) {
	time.Sleep(t)
	return "cached value", nil
}
```

### API Reference

#### `CachedFunc[K any, V any]`
A generic function type that can be wrapped with caching. `K` is the input parameter type, `V` is the result type.

#### `Config`
Defines cache configuration options:
- `TTL` (time.Duration): Time-to-live for each cache entry (default: 5 minutes)
- `Capacity` (int): Maximum number of cache entries (default: 1000)
- `CleanupInterval` (time.Duration): Interval for periodic cleanup (default: 1 minute)

#### `Hooks`
Provides optional hooks for cache lifecycle events and error logging. Hooks can be used for logging, metrics, tracing, or custom side effects. All hooks are optional and can be set individually.

**Available hooks:**

- `OnSet`: Called after a value is successfully stored in the cache (i.e., after a cache miss and successful function execution).
- `OnGet`: Called after a value is retrieved from the cache (cache hit).
- `OnExecute`: Called immediately before the underlying function is executed (i.e., on cache miss, before the function call).
- `OnDone`: Called after the underlying function finishes execution (regardless of success or error).
- `LogError`: Called whenever any other hook returns an error or panics, or when the underlying function panics or returns an error. This hook must never panic itself.

**Example: Logging with hooks**

```go
hooks := &fcache.Hooks{
    OnSet: func(arg any) error {
        fmt.Printf("[cache set] key: %v\n", arg)
        return nil
    },
    OnGet: func(arg any) error {
        fmt.Printf("[cache hit] key: %v\n", arg)
        return nil
    },
    OnExecute: func(arg any) error {
        fmt.Printf("[execute] key: %v\n", arg)
        return nil
    },
    OnDone: func(arg any) error {
        fmt.Printf("[done] key: %v\n", arg)
        return nil
    },
    LogError: func(err error) {
        log.Printf("[cache error] %v", err)
    },
}

cached := fcache.NewCachedFunction(fn, nil, hooks)
```

**When are hooks called?**
- `OnGet`: After a cache hit, with the input argument.
- `OnSet`: After a successful cache store (after a cache miss and successful function execution), with the input argument.
- `OnExecute`: Before the underlying function is called (on cache miss), with the input argument.
- `OnDone`: After the underlying function returns (on cache miss), with the input argument.
- `LogError`: Whenever any hook returns an error or panics, or when the underlying function panics or returns an error.

Hooks are always called safely: panics in hooks are caught and forwarded to `LogError` if set, and never propagate to the caller.

#### `NewCachedFunction
Wraps a function with a concurrent-safe caching layer.

```go
func NewCachedFunction[K any, V any](fn CachedFunc[K, V], opts *Config, hooks *Hooks) CachedFunc[K, V]
```
- `fn`: The function to cache. Must be of type `func(K) (V, error)`.
- `opts`: Optional cache configuration (TTL, capacity). Pass `nil` for defaults.
- `hooks`: Optional hooks for cache events. Pass `nil` if not needed.

Returns a function with the same signature as `fn`, but with caching applied.

---

## 🧪 Testing

fcache is thoroughly tested with unit tests covering:
- Correct value caching
- Result expiration after TTL
- Concurrent call deduplication
- Capacity limit and LRU eviction

To run all tests:

```sh
go test ./...
```

You will find tests in the `test/` directory, including:
- `correct_values_test.go`: Ensures values are cached and returned correctly
- `ttl_test.go`: Ensures results expire after the configured TTL
- `deduplicator_test.go`: Ensures concurrent calls with the same input are deduplicated
- `capacity_test.go`: Ensures the cache never exceeds its capacity and evicts the oldest entries

---

## 📈 Benchmarks

Benchmarks are provided in the `benchmark/` directory to compare:
- Direct function execution
- Cached execution (cold/warm)
- Performance under high concurrency

Run benchmarks with:

```sh
go test -bench=. ./benchmark
```


