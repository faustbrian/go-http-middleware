// Package observe emits one bounded transport completion event per request.
package observe

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/faustbrian/go-http-middleware/internal/httpx"
)

// Outcome is a bounded completion classification.
type Outcome string

const (
	// Success classifies responses below 400.
	Success Outcome = "success"
	// ClientError classifies responses from 400 through 499.
	ClientError Outcome = "client_error"
	// ServerError classifies responses at or above 500.
	ServerError Outcome = "server_error"
	// Canceled classifies requests whose context was canceled.
	Canceled Outcome = "canceled"
	// Panicked classifies handlers that propagated a panic.
	Panicked Outcome = "panicked"
)

// Event excludes raw paths, queries, headers, payloads, identities, and errors.
type Event struct {
	Method      string
	Route       string
	Status      int
	Bytes       int64
	Duration    time.Duration
	Proto       string
	Outcome     Outcome
	ClientClass string
}

// Policy injects observation without owning a logger or telemetry SDK.
type Policy struct {
	Observer        func(context.Context, Event)
	Route           func(*http.Request) string
	ClientClass     func(*http.Request) string
	Now             func() time.Time
	RepanicObserver bool
}

// ErrInvalidPolicy identifies invalid observation policy configuration.
var ErrInvalidPolicy = errors.New("observe: invalid policy")

// ConfigError reports an invalid observation policy field.
type ConfigError struct{ Field string }

func (e *ConfigError) Error() string { return fmt.Sprintf("observe: invalid %s", e.Field) }
func (e *ConfigError) Unwrap() error { return ErrInvalidPolicy }

// New constructs observation middleware.
func New(policy Policy) (func(http.Handler) http.Handler, error) {
	if policy.Observer == nil {
		return nil, &ConfigError{Field: "observer"}
	}
	if policy.Now == nil {
		policy.Now = time.Now
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := policy.Now()
			route := ""
			clientClass := ""
			trackedWriter, recorder := httpx.Track(w)
			defer func() {
				panicValue := recover()
				status := recorder.Status
				if status == 0 {
					status = http.StatusOK
				}
				result := outcome(r.Context(), status)
				if panicValue != nil {
					result = Panicked
				}
				event := Event{Method: r.Method, Route: route, Status: status, Bytes: recorder.Bytes, Duration: policy.Now().Sub(start), Proto: r.Proto, Outcome: result, ClientClass: clientClass}
				notify(r.Context(), policy, event)
				if panicValue != nil {
					panic(panicValue)
				}
			}()
			if policy.Route != nil {
				route = bounded(policy.Route(r), 128)
			}
			if policy.ClientClass != nil {
				clientClass = bounded(policy.ClientClass(r), 64)
			}
			next.ServeHTTP(trackedWriter, r)
		})
	}, nil
}

func outcome(ctx context.Context, status int) Outcome {
	if ctx.Err() != nil {
		return Canceled
	}
	if status >= 500 {
		return ServerError
	}
	if status >= 400 {
		return ClientError
	}
	return Success
}
func notify(ctx context.Context, policy Policy, event Event) {
	if policy.RepanicObserver {
		policy.Observer(ctx, event)
		return
	}
	defer func() { _ = recover() }()
	policy.Observer(ctx, event)
}
func bounded(value string, maximum int) string {
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}
