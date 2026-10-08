package cors_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/faustbrian/go-http-middleware/v2/cors"
)

func TestUnicodeOriginNormalizationPreservesAllowlistIdentity(t *testing.T) {
	t.Parallel()

	// These literal ASCII oracles are independently encoded NFC labels, not
	// values obtained from the IDNA implementation under test.
	for _, tc := range []struct {
		name      string
		origin    string
		canonical string
	}{
		{"supplementary rune", "https://\U00020061\u0308.example", "https://xn--ssa9981x.example"},
		{"Hangul then Latin", "https://\u1100\u1161a\u0300.example", "https://xn--0ca9278f.example"},
		{"composed Hangul then Latin", "https://\uac00\u00e0.example", "https://xn--0ca9278f.example"},
		{"intervening starter", "https://i\U000113c2\u0300\u0316.example", "https://xn--i-vbb9d9426m.example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, configured := range []string{tc.canonical, tc.origin} {
				middleware, err := cors.New(cors.Policy{AllowedOrigins: []string{configured}, AllowedMethods: []string{http.MethodGet}})
				if err != nil {
					t.Fatalf("New(%q): %v", configured, err)
				}
				for _, origin := range []string{tc.origin, tc.canonical} {
					for _, method := range []string{http.MethodOptions, http.MethodGet} {
						called := false
						req := httptest.NewRequest(method, "/", nil)
						req.Header.Set("Origin", origin)
						if method == http.MethodOptions {
							req.Header.Set("Access-Control-Request-Method", http.MethodGet)
						}
						recorder := httptest.NewRecorder()
						middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							called = true
							w.WriteHeader(http.StatusAccepted)
						})).ServeHTTP(recorder, req)
						wantStatus := http.StatusAccepted
						if method == http.MethodOptions {
							wantStatus = http.StatusNoContent
						}
						if recorder.Code != wantStatus || called != (method == http.MethodGet) || recorder.Header().Get("Access-Control-Allow-Origin") != tc.canonical {
							t.Fatalf("policy %q, %s origin %q: status=%d called=%v ACAO=%q; want status=%d ACAO=%q", configured, method, origin, recorder.Code, called, recorder.Header().Get("Access-Control-Allow-Origin"), wantStatus, tc.canonical)
						}
					}
				}
			}
		})
	}
}

func TestSupplementaryOriginCannotAliasUnrelatedAllowlistEntry(t *testing.T) {
	t.Parallel()

	middleware, err := cors.New(cors.Policy{AllowedOrigins: []string{"https://xn--4ca.example"}, AllowedMethods: []string{http.MethodGet}})
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodOptions, http.MethodGet} {
		called := false
		req := httptest.NewRequest(method, "/", nil)
		req.Header.Set("Origin", "https://\U00020061\u0308.example")
		if method == http.MethodOptions {
			req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		}
		recorder := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusAccepted)
		})).ServeHTTP(recorder, req)
		wantStatus := http.StatusAccepted
		if method == http.MethodOptions {
			wantStatus = http.StatusForbidden
		}
		if recorder.Code != wantStatus || called != (method == http.MethodGet) || recorder.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("%s: status=%d called=%v ACAO=%q", method, recorder.Code, called, recorder.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}

func TestInvalidJoinerOriginCannotReachWildcardGrant(t *testing.T) {
	t.Parallel()

	const invalid = "https://\u200d.example"
	if _, err := cors.New(cors.Policy{AllowedOrigins: []string{invalid}}); !errors.Is(err, cors.ErrInvalidPolicy) {
		t.Fatalf("invalid configured origin: %v", err)
	}
	middleware, err := cors.New(cors.Policy{AllowedOrigins: []string{"*"}, AllowedMethods: []string{http.MethodGet}})
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodOptions, http.MethodGet} {
		called := false
		req := httptest.NewRequest(method, "/", nil)
		req.Header.Set("Origin", invalid)
		if method == http.MethodOptions {
			req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		}
		recorder := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusAccepted)
		})).ServeHTTP(recorder, req)
		wantStatus := http.StatusAccepted
		if method == http.MethodOptions {
			wantStatus = http.StatusBadRequest
		}
		if recorder.Code != wantStatus || called != (method == http.MethodGet) || recorder.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("%s: status=%d called=%v ACAO=%q", method, recorder.Code, called, recorder.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}

func TestDynamicOriginReceivesOnlyValidCanonicalIdentity(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		origin string
		want   string
	}{
		{"https://\u200d.example", ""},
		{"https://\U00020061\u0308.example", "https://xn--ssa9981x.example"},
	} {
		seen := ""
		calls := 0
		middleware, err := cors.New(cors.Policy{AllowOrigin: func(_ context.Context, origin string) (bool, error) {
			calls++
			seen = origin
			return true, errors.New("authorization unavailable")
		}})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", tc.origin)
		recorder := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		})).ServeHTTP(recorder, req)
		wantCalls := 1
		if tc.want == "" {
			wantCalls = 0
		}
		if calls != wantCalls || seen != tc.want || recorder.Code != http.StatusAccepted || recorder.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("origin %q: calls=%d callback=%q status=%d ACAO=%q", tc.origin, calls, seen, recorder.Code, recorder.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}
