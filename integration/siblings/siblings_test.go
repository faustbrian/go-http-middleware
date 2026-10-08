package siblings_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/faustbrian/go-correlation"
	httpcorrelation "github.com/faustbrian/go-correlation/adapters/http"
	middleware "github.com/faustbrian/go-http-middleware/v2"
	"github.com/faustbrian/go-http-middleware/v2/adapter"
	"github.com/faustbrian/go-http-middleware/v2/observe"
	router "github.com/faustbrian/go-router/v2"
	"github.com/faustbrian/go-service/serverhttp"
)

func TestCorrelationHTTPIdentitySurvivesMiddlewareComposition(t *testing.T) {
	t.Parallel()
	uuidV4 := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	for _, tc := range []struct {
		name    string
		trusted bool
		invalid bool
	}{
		{name: "untrusted"},
		{name: "trusted", trusted: true},
		{name: "malformed", trusted: true, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			factory, err := correlation.NewFactory(correlation.FactoryOptions{})
			if err != nil {
				t.Fatal(err)
			}
			identity, err := httpcorrelation.New(factory, httpcorrelation.Options{
				Invalid: httpcorrelation.RejectInvalid,
				Trust:   func(*http.Request) bool { return tc.trusted },
			})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			var observed correlation.Values
			app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var ok bool
				observed, ok = correlation.FromContext(r.Context())
				if !ok || r.Header.Get("X-Correlation-ID") != observed.CorrelationID.String() ||
					r.Header.Get("X-Request-ID") != observed.RequestID.String() ||
					r.Header.Get("X-Causation-ID") != observed.CausationID.String() {
					t.Error("request headers and context identity differ")
				}
				w.WriteHeader(http.StatusNoContent)
			})
			chain, err := middleware.New(identity.Wrap)
			if err != nil {
				t.Fatal(err)
			}
			handler, err := chain.Handler(app)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("X-Correlation-ID", "workflow-1")
			request.Header.Set("X-Request-ID", "previous-hop")
			request.Header.Set("X-Causation-ID", "older-hop")
			if tc.invalid {
				request.Header.Set("X-Correlation-ID", "not valid")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if tc.invalid {
				if response.Code != http.StatusBadRequest || calls != 0 {
					t.Fatalf("malformed response=%d calls=%d", response.Code, calls)
				}
				return
			}
			if response.Code != http.StatusNoContent || calls != 1 {
				t.Fatalf("response=%d calls=%d", response.Code, calls)
			}
			if !uuidV4.MatchString(observed.RequestID.String()) {
				t.Fatalf("request ID is not canonical UUIDv4: %q", observed.RequestID)
			}
			if tc.trusted {
				if observed.CorrelationID != "workflow-1" || observed.CausationID != "previous-hop" {
					t.Fatalf("trusted identity=%+v", observed)
				}
			} else if !uuidV4.MatchString(observed.CorrelationID.String()) || observed.CausationID != "" {
				t.Fatalf("untrusted identity=%+v", observed)
			}
			if response.Header().Get("X-Correlation-ID") != observed.CorrelationID.String() ||
				response.Header().Get("X-Request-ID") != observed.RequestID.String() ||
				response.Header().Get("X-Causation-ID") != observed.CausationID.String() {
				t.Fatal("response headers and context identity differ")
			}
		})
	}
}

func TestGoRouterProvidesBoundedObservationMetadata(t *testing.T) {
	t.Parallel()
	builder := router.New()
	if err := builder.Register(router.Route{
		Name:    "users.show",
		Methods: []string{http.MethodGet},
		Path:    "/users/{id}",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		Middleware: []router.NamedMiddleware{{
			Name: "observe-route",
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
					matched, ok := router.MatchedRoute(request)
					if ok {
						observe.RecordRoute(request, matched.Name)
					}
					next.ServeHTTP(w, request)
				})
			},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	compiled, err := builder.Compile()
	if err != nil {
		t.Fatal(err)
	}
	var event observe.Event
	observer, _ := observe.New(observe.Policy{
		Observer: func(_ context.Context, value observe.Event) { event = value },
	})
	chain, _ := middleware.New(observer)
	handler, _ := chain.Handler(compiled)
	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/users/private-id", nil),
	)
	if event.Route != "users.show" {
		t.Fatalf("route = %q", event.Route)
	}
}

func TestGoServiceCoreOwnershipCannotBeInstalledTwice(t *testing.T) {
	t.Parallel()
	factory, err := correlation.NewFactory(correlation.FactoryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	requestIdentity, err := httpcorrelation.New(factory, httpcorrelation.Options{})
	if err != nil {
		t.Fatal(err)
	}
	bodyLimit, err := serverhttp.LimitBody(1024)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		concern adapter.Concern
		item    func(http.Handler) http.Handler
	}{
		{adapter.Recovery, serverhttp.Recover()},
		{adapter.RequestID, requestIdentity.Wrap},
		{adapter.BodyLimit, bodyLimit},
	} {
		descriptor, describeErr := adapter.Named(tc.concern, tc.item)
		if describeErr != nil {
			t.Fatal(describeErr)
		}
		chain, chainErr := middleware.Described(descriptor)
		if chainErr != nil {
			t.Fatal(chainErr)
		}
		if validationErr := adapter.ValidateGoService(chain, adapter.GoServiceDefaults()); !errors.Is(validationErr, adapter.ErrDuplicateOwnership) {
			t.Fatalf("%s validation error = %v", tc.concern, validationErr)
		}
	}
}
