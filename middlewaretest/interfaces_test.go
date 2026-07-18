package middlewaretest_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/faustbrian/go-http-middleware/observe"
	"github.com/faustbrian/go-http-middleware/recovery"
)

func TestTrackingWrappersPreserveExactOptionalInterfaces(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		middleware func(http.Handler) http.Handler
	}{
		{name: "observe", middleware: mustObserve(t)},
		{name: "recovery", middleware: mustRecovery(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			underlying := &allWriter{ResponseRecorder: httptest.NewRecorder()}
			called := false
			tc.middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				if _, ok := w.(http.Flusher); !ok {
					t.Error("Flusher missing")
				}
				if _, ok := w.(http.Hijacker); !ok {
					t.Error("Hijacker missing")
				}
				if _, ok := w.(http.Pusher); !ok {
					t.Error("Pusher missing")
				}
				if _, ok := w.(io.ReaderFrom); !ok {
					t.Error("ReaderFrom missing")
				}
			})).ServeHTTP(underlying, httptest.NewRequest(http.MethodGet, "/", nil))
			if !called {
				t.Fatal("handler not called")
			}

			plain := &plainWriter{header: make(http.Header)}
			tc.middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if _, ok := w.(http.Flusher); ok {
					t.Error("unexpected Flusher")
				}
				if _, ok := w.(http.Hijacker); ok {
					t.Error("unexpected Hijacker")
				}
				if _, ok := w.(http.Pusher); ok {
					t.Error("unexpected Pusher")
				}
				if _, ok := w.(io.ReaderFrom); ok {
					t.Error("unexpected ReaderFrom")
				}
			})).ServeHTTP(plain, httptest.NewRequest(http.MethodGet, "/", nil))
		})
	}
}

type plainWriter struct{ header http.Header }

func (w *plainWriter) Header() http.Header             { return w.header }
func (*plainWriter) Write(payload []byte) (int, error) { return len(payload), nil }
func (*plainWriter) WriteHeader(int)                   {}

type allWriter struct{ *httptest.ResponseRecorder }

func (w *allWriter) Flush() { w.ResponseRecorder.Flush() }
func (*allWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, http.ErrNotSupported
}
func (*allWriter) Push(string, *http.PushOptions) error { return http.ErrNotSupported }
func (w *allWriter) ReadFrom(reader io.Reader) (int64, error) {
	return io.Copy(w.ResponseRecorder, reader)
}

func mustObserve(t *testing.T) func(http.Handler) http.Handler {
	t.Helper()
	result, err := observe.New(observe.Policy{Observer: func(context.Context, observe.Event) {}})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func mustRecovery(t *testing.T) func(http.Handler) http.Handler {
	t.Helper()
	result, err := recovery.New(recovery.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
