// Package admission bounds local in-flight requests without implementing a
// distributed rate algorithm. New returns a factory whose uses share semaphore
// state. The caller owns the shutdown signal; middleware owns bounded wait timers
// and permit release, not active-handler cancellation, draining or server closure.
package admission

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/faustbrian/go-http-middleware/v2/internal/httpx"
)

// Policy configures bounded local request admission. Wait is capped at one
// minute.
type Policy struct {
	// MaxInFlight is required in [1,1000000]; capacity is shared by the returned factory.
	// MaxWaiters bounds queued callers in [0,1000000]; zero disables waiting.
	MaxInFlight, MaxWaiters int
	// Wait bounds queue residence up to one minute, not handler time; zero refuses saturation.
	Wait time.Duration
	// RetryAfterSeconds is a nonnegative refusal hint; zero omits the header.
	RetryAfterSeconds int
	// Shutdown is caller-owned; close to persistently refuse/wake waiters. It does not cancel or join admitted handlers.
	Shutdown <-chan struct{}
}

// ErrInvalidPolicy identifies invalid admission policy configuration.
var ErrInvalidPolicy = errors.New("admission: invalid policy")

// ConfigError reports an invalid admission policy field.
type ConfigError struct {
	// Field names the invalid policy field without echoing its configured value.
	Field string
}

// Error formats the policy-field diagnostic; constructors use fixed field labels.
func (e *ConfigError) Error() string { return fmt.Sprintf("admission: invalid %s", e.Field) }

// Unwrap exposes ErrInvalidPolicy for errors.Is.
func (e *ConfigError) Unwrap() error { return ErrInvalidPolicy }

const maximumWait = time.Minute

// New constructs semaphore admission. Channel wait order is not a contractual
// fairness guarantee; queues are bounded and allocate no waiter goroutine. Saturation
// or shutdown emits 503, canceled admission 408. Acquired permits survive until
// downstream return/panic; shutdown does not revoke them or join active handlers.
// A queued acquisition rechecks cancellation, but shutdown/acquisition arbitration
// is not a guarantee that no handler can start concurrently with signal closure.
func New(policy Policy) (func(http.Handler) http.Handler, error) {
	if policy.MaxInFlight < 1 || policy.MaxInFlight > 1_000_000 || policy.MaxWaiters < 0 || policy.MaxWaiters > 1_000_000 || policy.Wait < 0 || policy.Wait > maximumWait || policy.RetryAfterSeconds < 0 {
		return nil, &ConfigError{Field: "limit"}
	}
	permits := make(chan struct{}, policy.MaxInFlight)
	waiters := make(chan struct{}, policy.MaxWaiters)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if stopped(policy.Shutdown) {
				reject(w, policy)
				return
			}
			acquired := false
			select {
			case permits <- struct{}{}:
				acquired = true
			default:
			}
			if !acquired && policy.Wait != 0 && policy.MaxWaiters != 0 {
				select {
				case waiters <- struct{}{}:
					terminal := func() bool {
						defer func() { <-waiters }()
						timer := time.NewTimer(policy.Wait)
						defer timer.Stop()
						select {
						case permits <- struct{}{}:
							acquired = true
						case <-r.Context().Done():
							httpx.SafeError(w, http.StatusRequestTimeout, "request canceled\n")
							return true
						case <-policy.Shutdown:
							reject(w, policy)
							return true
						case <-timer.C:
						}
						return false
					}()
					if terminal {
						return
					}
				default:
				}
			}
			if !acquired {
				reject(w, policy)
				return
			}
			// A ready permit can win select concurrently with cancellation. Refuse
			// canceled execution and return that permit before invoking downstream.
			switch r.Context().Err() {
			case nil:
			default:
				<-permits
				httpx.SafeError(w, http.StatusRequestTimeout, "request canceled\n")
				return
			}
			defer func() { <-permits }()
			next.ServeHTTP(w, r)
		})
	}, nil
}

func stopped(shutdown <-chan struct{}) bool {
	if shutdown == nil {
		return false
	}
	select {
	case <-shutdown:
		return true
	default:
		return false
	}
}
func reject(w http.ResponseWriter, policy Policy) {
	if policy.RetryAfterSeconds != 0 {
		w.Header().Set("Retry-After", strconv.Itoa(policy.RetryAfterSeconds))
	}
	httpx.SafeError(w, http.StatusServiceUnavailable, "server busy\n")
}
