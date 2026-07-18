package deadline

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/faustbrian/go-http-middleware/internal/httpx"
)

var (
	// ErrHandlerTimeout is returned to late writes after the response timeout.
	ErrHandlerTimeout = errors.New("deadline: handler timeout")
	// ErrResponseTooLarge identifies output above the configured buffer bound.
	ErrResponseTooLarge = errors.New("deadline: response too large")
)

// TimeoutPolicy configures a bounded non-streaming handler timeout. A handler
// that ignores cancellation may continue executing, but cannot write through
// the closed buffer or retain middleware-owned unbounded response memory.
type TimeoutPolicy struct {
	Timeout          time.Duration
	MaxResponseBytes int
	Status           int
}

// NewTimeout constructs bounded buffered timeout middleware. The wrapped
// writer intentionally exposes no streaming, hijacking, push, or full-duplex
// capability because those operations are incompatible with response replay.
func NewTimeout(policy TimeoutPolicy) (func(http.Handler) http.Handler, error) {
	if policy.Status == 0 {
		policy.Status = http.StatusServiceUnavailable
	}
	if policy.Timeout <= 0 || policy.MaxResponseBytes < 1 || policy.MaxResponseBytes > 16<<20 || policy.Status < 500 || policy.Status > 599 {
		return nil, &ConfigError{Field: "timeout response policy"}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), policy.Timeout)
			defer cancel()
			buffer := newTimeoutWriter(policy.MaxResponseBytes)
			completed := make(chan any, 1)
			go func() {
				var panicValue any
				defer func() { panicValue = recover(); completed <- panicValue }()
				next.ServeHTTP(buffer, r.WithContext(ctx))
			}()
			select {
			case panicValue := <-completed:
				header, status, payload, overflow := buffer.finish()
				if panicValue != nil {
					panic(panicValue)
				}
				if overflow {
					httpx.SafeError(w, http.StatusInternalServerError, "internal server error\n")
					return
				}
				copyHeaders(w.Header(), header)
				w.WriteHeader(status)
				_, _ = w.Write(payload)
			case <-ctx.Done():
				buffer.timeout()
				httpx.SafeError(w, policy.Status, "handler timeout\n")
			}
		})
	}, nil
}

type timeoutWriter struct {
	mu       sync.Mutex
	header   http.Header
	status   int
	payload  bytes.Buffer
	maximum  int
	closed   bool
	overflow bool
}

func newTimeoutWriter(maximum int) *timeoutWriter {
	return &timeoutWriter{header: make(http.Header), maximum: maximum}
}
func (w *timeoutWriter) Header() http.Header { return w.header }
func (w *timeoutWriter) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed && w.status == 0 && status >= 200 {
		w.status = status
	}
}
func (w *timeoutWriter) Write(payload []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, ErrHandlerTimeout
	}
	if w.overflow {
		return 0, ErrResponseTooLarge
	}
	if w.payload.Len()+len(payload) > w.maximum {
		w.overflow = true
		return 0, ErrResponseTooLarge
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.payload.Write(payload)
}
func (w *timeoutWriter) timeout() { w.mu.Lock(); w.closed = true; w.mu.Unlock() }
func (w *timeoutWriter) finish() (http.Header, int, []byte, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	return w.header.Clone(), status, append([]byte(nil), w.payload.Bytes()...), w.overflow
}
func copyHeaders(destination, source http.Header) {
	for name := range destination {
		destination.Del(name)
	}
	for name, values := range source {
		destination[name] = append([]string(nil), values...)
	}
}
