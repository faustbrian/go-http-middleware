package compress

import (
	"compress/gzip"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfigurationAndNegotiationBoundaries(t *testing.T) {
	t.Parallel()
	if _, err := New(Policy{}); err != nil {
		t.Fatalf("default policy error = %v", err)
	}
	for _, policy := range []Policy{
		{MinimumBytes: -1}, {MinimumBytes: 2, MaxBuffer: 1}, {MinimumBytes: 1, MaxBuffer: 16<<20 + 1},
		{MinimumBytes: 1, MaxBuffer: 1, Level: gzip.HuffmanOnly - 1}, {MinimumBytes: 1, MaxBuffer: 1, Level: gzip.BestCompression + 1},
		{MinimumBytes: 1, MaxBuffer: 1, MaxHeaderBytes: -1}, {MinimumBytes: 1, MaxBuffer: 1, MaxHeaderBytes: 1<<20 + 1},
		{MinimumBytes: 1, MaxBuffer: 1, ExcludedTypes: make([]string, 65)},
	} {
		_, err := New(policy)
		var configuration *ConfigError
		if !errors.As(err, &configuration) || !errors.Is(err, ErrInvalidPolicy) || configuration.Error() == "" {
			t.Fatalf("New(%+v) error = %v", policy, err)
		}
	}
	for _, lines := range [][]string{{"gzip;"}, {"gzip;level=1"}, {"gzip;q=1;q=1"}, {"gzip;q=no"}, {strings.Repeat("x", 9)}} {
		if _, _, ok := negotiate(lines, 8); ok {
			t.Fatalf("negotiate(%q) succeeded", lines)
		}
	}
	for _, tc := range []struct {
		lines          []string
		gzip, identity float64
		ok             bool
	}{
		{nil, 0, 1, true}, {[]string{""}, 0, 1, true}, {[]string{"*;q=0.5"}, .5, 1, true},
		{[]string{"*;q=0"}, 0, 0, false}, {[]string{"gzip;q=0.5, gzip;q=0.8, identity;q=0"}, .8, 0, true},
		{[]string{"br, identity;q=0"}, 0, 0, false},
	} {
		gzipQ, identityQ, ok := negotiate(tc.lines, 128)
		if gzipQ != tc.gzip || identityQ != tc.identity || ok != tc.ok {
			t.Fatalf("negotiate(%q) = %v, %v, %v", tc.lines, gzipQ, identityQ, ok)
		}
	}
}

func TestResponseBufferAndCompressionDecisionMatrix(t *testing.T) {
	t.Parallel()
	destination := httptest.NewRecorder()
	buffer := newBuffer(destination, 2)
	buffer.Header().Set("X-Test", "yes")
	buffer.WriteHeader(http.StatusEarlyHints)
	if destination.Code != http.StatusEarlyHints {
		t.Fatalf("informational status = %d", destination.Code)
	}
	destination = httptest.NewRecorder()
	buffer = newBuffer(destination, 2)
	buffer.Header().Set("X-Test", "yes")
	buffer.WriteHeader(http.StatusCreated)
	buffer.WriteHeader(http.StatusAccepted)
	if _, err := buffer.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := buffer.Write([]byte("d")); err != nil {
		t.Fatal(err)
	}
	buffer.commitIdentity()
	if !buffer.spilled || destination.Code != http.StatusCreated || destination.Body.String() != "abcd" {
		t.Fatalf("buffer = %+v, response = %d %q", buffer, destination.Code, destination.Body.String())
	}

	baseRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	makeBuffer := func(status int, header http.Header, body string) *responseBuffer {
		value := newBuffer(httptest.NewRecorder(), 100)
		value.status = status
		value.header = header
		_, _ = value.buffer.WriteString(body)
		return value
	}
	for _, tc := range []struct {
		name             string
		request          *http.Request
		buffer           *responseBuffer
		gzipQ, identityQ float64
		excluded         []string
	}{
		{"gzip disabled", baseRequest, makeBuffer(200, http.Header{}, "body"), 0, 1, nil},
		{"identity preferred", baseRequest, makeBuffer(200, http.Header{}, "body"), .5, 1, nil},
		{"head", httptest.NewRequest(http.MethodHead, "/", nil), makeBuffer(200, http.Header{}, "body"), 1, 1, nil},
		{"informational", baseRequest, makeBuffer(101, http.Header{}, "body"), 1, 1, nil},
		{"no content", baseRequest, makeBuffer(204, http.Header{}, "body"), 1, 1, nil},
		{"not modified", baseRequest, makeBuffer(304, http.Header{}, "body"), 1, 1, nil},
		{"request range", requestWithHeader("Range", "bytes=0-1"), makeBuffer(200, http.Header{}, "body"), 1, 1, nil},
		{"response range", baseRequest, makeBuffer(200, http.Header{"Content-Range": {"bytes 0-1/4"}}, "body"), 1, 1, nil},
		{"encoded", baseRequest, makeBuffer(200, http.Header{"Content-Encoding": {"br"}}, "body"), 1, 1, nil},
		{"no transform", baseRequest, makeBuffer(200, http.Header{"Cache-Control": {"private, NO-TRANSFORM"}}, "body"), 1, 1, nil},
		{"small", baseRequest, makeBuffer(200, http.Header{}, "x"), 1, 1, nil},
		{"excluded", baseRequest, makeBuffer(200, http.Header{"Content-Type": {"image/png"}}, "body"), 1, 1, []string{"IMAGE/PNG"}},
	} {
		if shouldCompress(tc.request, tc.buffer, tc.gzipQ, tc.identityQ, 2, tc.excluded) {
			t.Fatalf("%s compressed", tc.name)
		}
	}
	if !shouldCompress(baseRequest, makeBuffer(0, http.Header{"Content-Type": {"text/plain"}}, "body"), 1, 1, 2, nil) {
		t.Fatal("eligible response not compressed")
	}
	if statusOrOK(0) != http.StatusOK || statusOrOK(201) != 201 {
		t.Fatal("status default failed")
	}
}

func TestMiddlewareRejectsNoAcceptableCodingAndSpills(t *testing.T) {
	t.Parallel()
	middleware, err := New(Policy{MinimumBytes: 1, MaxBuffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "identity;q=0, gzip;q=0")
	recorder := httptest.NewRecorder()
	middleware(http.NotFoundHandler()).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotAcceptable {
		t.Fatalf("status = %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder = httptest.NewRecorder()
	middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("large")) })).ServeHTTP(recorder, request)
	if recorder.Header().Get("Content-Encoding") != "" || recorder.Body.String() != "large" {
		t.Fatalf("response = %v %q", recorder.Header(), recorder.Body.String())
	}
}

func requestWithHeader(name, value string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(name, value)
	return request
}
