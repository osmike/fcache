// Package keygen provides utilities for generating deterministic cache keys
//
// based on input values. It handles various data types, encodes them,
// and ensures that keys are consistent and manageable in size.
// It supports hashing for long strings and complex types to maintain a
package keygen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/osmike/fcache/internal/lib/errs"
)

// Maximum length for string keys before hashing
const maxLen = 100

var (
	// ErrMarshallJSON indicates a failure to marshal a value to JSON.
	ErrMarshallJSON = fmt.Errorf("error marshalling to JSON")

	// ErrBuildKey indicates a failure to build a cache key from a value.
	ErrBuildKey = fmt.Errorf("error building cache key")
)

// BuildKey returns a deterministic string key for caching based on the provided value.
//
//   - value: Any value to be encoded as a cache key. Supports primitives, strings, fmt.Stringer, slices, maps, structs, etc.
//
// The key is deterministic for the same input value. If the encoded key exceeds maxLen, it is hashed to ensure a consistent length.
// Returns an error if the value cannot be encoded.
func BuildKey(value any) (string, error) {
	encoded, err := encodeValue(value)
	if err != nil {
		return "", errs.NewError(ErrBuildKey, map[string]interface{}{
			"operation": "building cache key",
			"value":     value,
			"error":     err,
		})
	}
	if len(encoded) > maxLen {
		// If the concatenated string is too long, hash it to ensure a consistent key
		return hashBytes([]byte(encoded)), nil
	}

	return encoded, nil
}

// encodeValue encodes a single value into a string suitable for use as a cache key.
//
// Handles primitive types, strings, fmt.Stringer, and complex types (slices, maps, structs).
// For context.Context, returns a placeholder string.
// If the encoded string is too long, it is hashed.
// Returns an error if encoding fails.
func encodeValue(v interface{}) (string, error) {
	switch val := v.(type) {
	// Primitive types and basic values
	case nil:
		return "nil", nil

	case context.Context:
		// For context, we return a placeholder since contexts are not serializable
		return "context", nil

	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64:
		return fmt.Sprint(val), nil

	case bool:
		return "b:" + fmt.Sprint(val), nil

	case string:
		return encodeString("s:" + val)

	case fmt.Stringer:
		s := val.String()
		return encodeString("s:" + s)

	// Collections and complex types
	default:
		return encodeComplex(val)
	}
}

// encodeString encodes a string value for use as a cache key.
//
// If the string exceeds maxLen, it is hashed to ensure a consistent key length.
// Otherwise, returns the string as is.
func encodeString(s string) (string, error) {
	if len(s) > maxLen {
		return hashBytes([]byte(s)), nil
	}
	return s, nil
}

// encodeComplex encodes complex types (slices, maps, structs) for use as a cache key.
//
// Marshals the value to JSON. For maps, always hashes the JSON to ignore key order.
// For slices/arrays, hashes if the JSON is too long. For other types, returns the JSON string directly if short enough.
// Returns an error if marshaling fails.
func encodeComplex(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", errs.NewError(ErrMarshallJSON, map[string]interface{}{
			"operation": "encoding complex value to build cache key",
			"value":     v,
			"error":     err,
		})
	}

	switch v.(type) {
	case map[string]interface{}:
		// for maps, we hash the JSON to ignore key order
		return hashBytes(data), nil
	default:
		// for slices, arrays, and other types
		if shouldHashData(data) {
			return hashBytes(data), nil
		}
		// for other types, return the JSON string directly
		return string(data), nil
	}
}

// shouldHashData returns true if the JSON representation of a value is too long for a cache key.
func shouldHashData(data []byte) bool {
	return len(data) > maxLen
}

// hashBytes hashes the byte slice using SHA-256 and returns the hex string.
func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
