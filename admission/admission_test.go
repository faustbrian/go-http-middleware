package admission_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/faustbrian/go-http-middleware/v2/admission"
)

func TestImmediateAdmissionRejectsAboveLimitAndReleasesPermit(t *testing.T) {
	t.Parallel()

	middleware, err := admission.New(admission.Policy{MaxInFlight: 1})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		close(done)
	}()
	<-entered
	overloaded := httptest.NewRecorder()
	handler.ServeHTTP(overloaded, httptest.NewRequest(http.MethodGet, "/", nil))
	if overloaded.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", overloaded.Code)
	}
	if retryAfter := overloaded.Header().Get("Retry-After"); retryAfter != "" {
		t.Fatalf("Retry-After = %q, want omitted", retryAfter)
	}
	close(release)
	<-done
}

func TestBoundedWaitHonorsCancellationWithoutLeakingPermit(t *testing.T) {
	t.Parallel()

	middleware, err := admission.New(admission.Policy{MaxInFlight: 1, MaxWaiters: 1, Wait: time.Second})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	block := make(chan struct{})
	entered := make(chan struct{}, 1)
	done := make(chan struct{})
	var release sync.Once
	t.Cleanup(func() {
		release.Do(func() { close(block) })
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("admitted holder did not finish during cleanup")
		}
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-block
		w.WriteHeader(http.StatusNoContent)
	}))
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		close(done)
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not enter the admitted handler")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx))
	if recorder.Code != http.StatusRequestTimeout {
		t.Fatalf("status = %d", recorder.Code)
	}
	release.Do(func() { close(block) })
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("admitted holder did not finish after release")
	}
	reused := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(reused, httptest.NewRequest(http.MethodGet, "/", nil))
	if reused.Code != http.StatusNoContent {
		t.Fatalf("status after cancellation and holder release = %d, want 204", reused.Code)
	}
}

func TestShutdownRejectsNewAndWaitingAdmissions(t *testing.T) {
	t.Parallel()

	shutdown := make(chan struct{})
	close(shutdown)
	middleware, _ := admission.New(admission.Policy{MaxInFlight: 1, MaxWaiters: 1, Wait: time.Second, Shutdown: shutdown})
	recorder := httptest.NewRecorder()
	middleware(http.NotFoundHandler()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", recorder.Code)
	}
}
