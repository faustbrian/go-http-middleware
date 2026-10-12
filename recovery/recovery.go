// Package recovery contains downstream panics on the serving goroutine below
// an explicit New boundary, not panics in arbitrary spawned goroutines. It owns
// no logger or shutdown. Optional stack traces are sensitive caller-owned
// observations; fixed responses do not expose panic values.
package recovery

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/faustbrian/go-http-middleware/v2/internal/httpx"
)

// Class is deliberately bounded and never includes the panic value.
type Class string

const (
	// ApplicationPanic classifies non-runtime application panics.
	ApplicationPanic Class = "application_panic"
	// RuntimePanic classifies values implementing runtime.Error.
	RuntimePanic Class = "runtime_panic"
)

// Event reports safe panic classification and response ownership state.
type Event struct {
	// Class is a bounded application/runtime classification, not the panic value.
	Class Class
	// Committed records observed writer commitment or successful takeover at recovery.
	Committed bool
	// Stack is an owned optional current-goroutine trace; it may reveal paths and runtime details and is not redacted.
	Stack []byte
}

// Policy configures recovery observation. Observer panics are contained.
type Policy struct {
	// Observer is borrowed synchronous caller code; panics are contained, latency/concurrent safety are not supplied.
	Observer func(Event)
	// CaptureStack opts into a bounded runtime.Stack trace, which requires caller privacy controls.
	CaptureStack bool
	// MaxStackBytes defaults to 65536 when capture is enabled; maximum 1 MiB, otherwise must be zero.
	MaxStackBytes int
}

// ErrInvalidPolicy identifies invalid recovery policy configuration.
var ErrInvalidPolicy = errors.New("recovery: invalid policy")

const (
	defaultMaxStackBytes = 65_536
	maximumMaxStackBytes = 1_048_576
)

// ConfigError reports an invalid recovery policy field.
type ConfigError struct {
	// Field names the invalid policy field without echoing its configured value.
	Field string
}

// Error formats the policy-field diagnostic; constructors use fixed field labels.
func (e *ConfigError) Error() string { return fmt.Sprintf("recovery: invalid %s", e.Field) }

// Unwrap exposes ErrInvalidPolicy for errors.Is.
func (e *ConfigError) Unwrap() error { return ErrInvalidPolicy }

// New constructs recovery middleware. http.ErrAbortHandler is re-panicked so
// normal connection abort behavior remains owned by net/http. Other panics notify
// the borrowed observer synchronously, then emit 500 only while uncommitted.
// Committed/taken-over output is not rewritten and the original application panic
// is swallowed. Observer panics are contained, not blocking or privacy leakage.
func New(policy Policy) (func(http.Handler) http.Handler, error) {
	if policy.CaptureStack && policy.MaxStackBytes == 0 {
		policy.MaxStackBytes = defaultMaxStackBytes
	}
	if policy.MaxStackBytes < 0 || policy.MaxStackBytes > maximumMaxStackBytes || (!policy.CaptureStack && policy.MaxStackBytes != 0) {
		return nil, &ConfigError{Field: "stack limit"}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trackedWriter, recorder := httpx.Track(w)
			defer func() {
				panicValue := recover()
				if panicValue == nil {
					return
				}
				if panicErr, ok := panicValue.(error); ok && errors.Is(panicErr, http.ErrAbortHandler) {
					panic(panicValue)
				}
				class := ApplicationPanic
				if _, ok := panicValue.(runtime.Error); ok {
					class = RuntimePanic
				}
				event := Event{Class: class, Committed: recorder.Committed}
				if policy.CaptureStack {
					stack := make([]byte, policy.MaxStackBytes)
					event.Stack = stack[:runtime.Stack(stack, false)]
				}
				observe(policy.Observer, event)
				if !recorder.Committed {
					for name := range w.Header() {
						w.Header().Del(name)
					}
					httpx.SafeError(trackedWriter, http.StatusInternalServerError, "internal server error\n")
				}
			}()
			next.ServeHTTP(trackedWriter, r)
		})
	}, nil
}

func observe(observer func(Event), event Event) {
	if observer == nil {
		return
	}
	defer func() { _ = recover() }()
	observer(event)
}
