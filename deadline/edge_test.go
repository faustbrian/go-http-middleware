package deadline

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConfigurationErrorContractAndTimeoutBounds(t *testing.T) {
	t.Parallel()
	_, err := New(Policy{})
	var configuration *ConfigError
	if !errors.As(err, &configuration) || !errors.Is(err, ErrInvalidPolicy) || configuration.Error() == "" {
		t.Fatalf("New() error = %v", err)
	}
	for _, policy := range []TimeoutPolicy{{}, {Timeout: time.Second}, {Timeout: time.Second, MaxResponseBytes: 16<<20 + 1}, {Timeout: time.Second, MaxResponseBytes: 1, Status: 499}, {Timeout: time.Second, MaxResponseBytes: 1, Status: 600}} {
		if _, err := NewTimeout(policy); !errors.Is(err, ErrInvalidPolicy) {
			t.Fatalf("NewTimeout(%+v) error = %v", policy, err)
		}
	}
}

func TestTimeoutWriterStateMachine(t *testing.T) {
	t.Parallel()
	w := newTimeoutWriter(2)
	w.Header().Set("X-Test", "yes")
	w.WriteHeader(http.StatusEarlyHints)
	w.WriteHeader(http.StatusCreated)
	w.WriteHeader(http.StatusAccepted)
	if _, err := w.Write([]byte("abc")); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("overflow error = %v", err)
	}
	if _, err := w.Write([]byte("a")); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("repeat overflow error = %v", err)
	}
	header, status, payload, overflow := w.finish()
	if header.Get("X-Test") != "yes" || status != http.StatusCreated || len(payload) != 0 || !overflow {
		t.Fatalf("finish = %v, %d, %q, %v", header, status, payload, overflow)
	}
	if _, err := w.Write([]byte("a")); !errors.Is(err, ErrHandlerTimeout) {
		t.Fatalf("late write error = %v", err)
	}

	empty := newTimeoutWriter(1)
	_, status, _, _ = empty.finish()
	if status != http.StatusOK {
		t.Fatalf("empty status = %d", status)
	}
	implicit := newTimeoutWriter(1)
	if count, err := implicit.Write([]byte("a")); count != 1 || err != nil {
		t.Fatalf("write = %d, %v", count, err)
	}
}

func TestTimeoutPropagatesPanicAndReplacesHeaders(t *testing.T) {
	t.Parallel()
	middleware, err := NewTimeout(TimeoutPolicy{Timeout: time.Second, MaxResponseBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	recorder.Header().Set("X-Old", "remove")
	middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Header().Set("X-New", "keep") })).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Header().Get("X-Old") != "" || recorder.Header().Get("X-New") != "keep" {
		t.Fatalf("headers = %v", recorder.Header())
	}

	defer func() {
		if recover() != "boom" {
			t.Fatal("handler panic not propagated")
		}
	}()
	middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}
