// Package observe observes transport completion with New and optional RecordRoute.
// It owns no logger, exporter or callback worker. Completion calls are synchronous,
// not delivery acknowledgment; caller clocks/classifiers/observers own privacy,
// latency, concurrency and lifecycle. Panicking clocks can prevent observation.
package observe

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/faustbrian/go-http-middleware/v2/internal/httpx"
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

// Event does not automatically collect paths, queries, headers, payloads,
// identities or errors. Caller-supplied Route/ClientClass can nevertheless leak
// them; byte truncation is neither sanitization nor bounded cardinality. Method
// and Proto use fixed known values plus OTHER.
type Event struct {
	// Method is a known HTTP method or OTHER.
	Method string
	// Route is caller-selected metadata truncated to 128 bytes; no privacy/cardinality sanitization is applied.
	Route string
	// Status is recorded final status or 200 when absent; informational responses alone do not commit.
	Status int
	// Bytes counts successful writer payload bytes, not wire framing or guaranteed delivery.
	Bytes int64
	// Duration uses the injected clock and is clamped to zero when negative.
	Duration time.Duration
	// Proto is a known HTTP version or OTHER.
	Proto string
	// Outcome gives panic precedence over cancellation, then status classification.
	Outcome Outcome
	// ClientClass is caller-selected metadata truncated to 64 bytes, potentially splitting UTF-8.
	ClientClass string
}

// Policy injects observation without owning a logger or telemetry SDK. Route
// and ClientClass run once at completion; their panics are contained.
type Policy struct {
	// Observer is required borrowed synchronous caller code; receives the completion request context, possibly canceled.
	Observer func(context.Context, Event)
	// Route is a fallback classification extractor after downstream completion; panic yields empty metadata.
	Route func(*http.Request) string
	// ClientClass is a completion classification extractor; panic yields empty metadata.
	ClientClass func(*http.Request) string
	// Now defaults to time.Now; borrowed clock panics propagate and can prevent event emission.
	Now func() time.Time
	// RepanicObserver lets observer panics propagate, possibly masking an original handler panic; zero contains them.
	RepanicObserver bool
}

type routeContextKey struct{}

type routeState struct {
	mu    sync.RWMutex
	value string
}

// ErrInvalidPolicy identifies invalid observation policy configuration.
var ErrInvalidPolicy = errors.New("observe: invalid policy")

// ConfigError reports an invalid observation policy field.
type ConfigError struct {
	// Field names the invalid policy field without echoing its configured value.
	Field string
}

// Error formats the policy-field diagnostic; constructors use fixed field labels.
func (e *ConfigError) Error() string { return fmt.Sprintf("observe: invalid %s", e.Field) }

// Unwrap exposes ErrInvalidPolicy for errors.Is.
func (e *ConfigError) Unwrap() error { return ErrInvalidPolicy }

// New attempts synchronous observation after downstream completion, including
// propagated panic/cancellation. Extractor panics produce empty classifications;
// observer panics are contained unless RepanicObserver opts out. Clock panics
// are not contained. Callback implementations are borrowed and may run concurrently
// across requests; no callback deadline or network/telemetry delivery is provided.
func New(policy Policy) (func(http.Handler) http.Handler, error) {
	if policy.Observer == nil {
		return nil, &ConfigError{Field: "observer"}
	}
	if policy.Now == nil {
		policy.Now = time.Now
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := &routeState{}
			r = r.WithContext(context.WithValue(r.Context(), routeContextKey{}, route))
			start := policy.Now()
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
				duration := max(policy.Now().Sub(start), 0)
				routeName := route.load()
				if routeName == "" {
					routeName = metadata(policy.Route, r, 128)
				}
				clientClass := metadata(policy.ClientClass, r, 64)
				event := Event{Method: methodClass(r.Method), Route: routeName, Status: status, Bytes: recorder.Bytes, Duration: duration, Proto: protocolClass(r.Proto), Outcome: result, ClientClass: clientClass}
				notify(r.Context(), policy, event)
				if panicValue != nil {
					panic(panicValue)
				}
			}()
			next.ServeHTTP(trackedWriter, r)
		})
	}, nil
}

// RecordRoute stores a bounded route classification for the outer observation
// layer. Route-local middleware may call it after a router has matched. It
// returns false when request is nil or observation middleware is absent. Last
// recorded value wins under the request-scoped lock; truncation is by bytes and
// can split UTF-8. The route is not validated or sanitized; use trusted labels.
func RecordRoute(request *http.Request, route string) bool {
	if request == nil {
		return false
	}
	state, ok := request.Context().Value(routeContextKey{}).(*routeState)
	if !ok {
		return false
	}
	state.mu.Lock()
	state.value = bounded(route, 128)
	state.mu.Unlock()
	return true
}

func (state *routeState) load() string {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.value
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
	return value[:min(len(value), maximum)]
}
func metadata(extractor func(*http.Request) string, request *http.Request, maximum int) (value string) {
	if extractor == nil {
		return ""
	}
	defer func() { _ = recover() }()
	return bounded(extractor(request), maximum)
}
func methodClass(method string) string {
	switch method {
	case http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodHead,
		http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut,
		http.MethodTrace:
		return method
	default:
		return "OTHER"
	}
}
func protocolClass(protocol string) string {
	switch protocol {
	case "HTTP/1.0", "HTTP/1.1", "HTTP/2.0", "HTTP/3.0":
		return protocol
	default:
		return "OTHER"
	}
}
