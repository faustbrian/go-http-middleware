// Package deadline offers synchronous context deadlines with New and bounded
// non-streaming response timeouts with NewTimeout. Only the latter starts handler
// workers; timed-out workers retain capacity until cooperative return. Neither
// owns server read/write/idle, upstream, shutdown or process limits.
package deadline

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrInvalidPolicy identifies invalid deadline policy configuration.
var ErrInvalidPolicy = errors.New("deadline: invalid policy")

// ConfigError reports an invalid deadline policy field.
type ConfigError struct {
	// Field names the invalid policy field without echoing its configured value.
	Field string
}

// Error formats the policy-field diagnostic; constructors use fixed field labels.
func (e *ConfigError) Error() string { return fmt.Sprintf("deadline: invalid %s", e.Field) }

// Unwrap exposes ErrInvalidPolicy for errors.Is.
func (e *ConfigError) Unwrap() error { return ErrInvalidPolicy }

// Policy configures a handler context timeout of at most 24 hours. Code that
// ignores its context is not interrupted.
type Policy struct {
	// Timeout must be positive and at most 24 hours; it only signals the request context.
	Timeout time.Duration
}

const maximumTimeout = 24 * time.Hour

// New constructs synchronous context-deadline middleware. A shorter parent
// deadline is preserved; cancellation is deferred until downstream returns. The
// request clone shares Body and Header with its parent. No timeout response,
// goroutine, writer buffer or forced interruption is introduced; panics propagate.
func New(policy Policy) (func(http.Handler) http.Handler, error) {
	if policy.Timeout <= 0 || policy.Timeout > maximumTimeout {
		return nil, &ConfigError{Field: "timeout"}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), policy.Timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}
