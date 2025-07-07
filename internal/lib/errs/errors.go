package errs

import "fmt"

// NewError wraps an error with additional context fields for structured error reporting.
//
//   - err: The base error to wrap.
//   - fields: A map of key-value pairs providing additional context.
//
// Returns an error that includes both the original error and the provided fields.
func NewError(errType error, kv map[string]interface{}) error {
	if kv == nil {
		return fmt.Errorf("[fcache error], [%w]", errType)
	}
	var details string
	for k, v := range kv {
		switch val := v.(type) {
		case error:
			details += fmt.Sprintf("%s: %v; ", k, val.Error())
		default:
			details += fmt.Sprintf("%s: %v; ", k, val)
		}
	}
	return fmt.Errorf("[fcache error], [%w], details: [%s]", errType, details)
}
