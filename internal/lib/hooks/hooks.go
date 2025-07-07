// package hooks

package hooks

import (
	"fmt"
)

// HookFunc is called on lifecycle events. It receives any number of arguments
// and may return an error to signal that something went wrong.
type HookFunc func(arg any) error

// HookFuncError is called whenever another hook errors or panics.
// It must never panic itself.
type HookFuncError func(err error)

// Hooks holds the set of lifecycle hooks and an error‐logging hook.
type Hooks struct {
	OnSet     HookFunc      // called after a Set operation
	OnGet     HookFunc      // called after a Get operation
	OnExecute HookFunc      // called after a function execution
	OnDone    HookFunc      // called after a function execution is done
	LogError  HookFuncError // called on any hook error or panic
}

// Run executes the given hook fn with the provided args.
// If fn returns an error *or* panics, Run will recover and forward
// the error to Hooks.LogError (if non‐nil), and will not panic itself.
func (h *Hooks) Run(fn HookFunc, arg any) {
	if fn == nil {
		return
	}

	// catch panics in the hook
	defer func() {
		if r := recover(); r != nil {
			h.safeLogError(toError(r))
		}
	}()

	// run the hook
	if err := fn(arg); err != nil {
		h.safeLogError(err)
	}
}

// safeLogError calls the LogError hook if set, and recovers if it panics.
func (h *Hooks) safeLogError(err error) {
	if h.LogError == nil {
		return
	}
	defer func() {
		recover() // swallow any panic in LogError
	}()
	h.LogError(err)
}

// toError converts a recovered panic value into an error.
func toError(r any) error {
	switch v := r.(type) {
	case error:
		return v
	case string:
		return fmt.Errorf("%s", v)
	default:
		return fmt.Errorf("%v", v)
	}
}
