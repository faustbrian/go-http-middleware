// Package compress performs bounded gzip content-coding negotiation. Sensitive
// dynamic responses should opt out by setting Cache-Control: no-transform.
package compress

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/faustbrian/go-http-middleware/internal/httpx"
)

// Policy configures bounded gzip response compression.
type Policy struct {
	MinimumBytes, MaxBuffer int
	Level, MaxHeaderBytes   int
	ExcludedTypes           []string
}

// ErrInvalidPolicy identifies invalid compression policy configuration.
var ErrInvalidPolicy = errors.New("compress: invalid policy")

// ConfigError reports an invalid compression policy field.
type ConfigError struct{ Field string }

func (e *ConfigError) Error() string { return fmt.Sprintf("compress: invalid %s", e.Field) }
func (e *ConfigError) Unwrap() error { return ErrInvalidPolicy }

// New constructs gzip middleware. Responses are buffered up to MaxBuffer;
// larger responses spill as identity, so retained memory remains bounded.
func New(policy Policy) (func(http.Handler) http.Handler, error) {
	if policy.MinimumBytes == 0 {
		policy.MinimumBytes = 1024
	}
	if policy.MaxBuffer == 0 {
		policy.MaxBuffer = 1 << 20
	}
	if policy.Level == 0 {
		policy.Level = gzip.DefaultCompression
	}
	if policy.MaxHeaderBytes == 0 {
		policy.MaxHeaderBytes = 8192
	}
	if policy.MinimumBytes < 1 || policy.MaxBuffer < policy.MinimumBytes ||
		policy.MaxBuffer > 16<<20 || policy.Level < gzip.HuffmanOnly ||
		policy.Level > gzip.BestCompression || policy.MaxHeaderBytes < 1 ||
		policy.MaxHeaderBytes > 1<<20 || len(policy.ExcludedTypes) > 64 {
		return nil, &ConfigError{Field: "limit"}
	}
	excluded := append([]string(nil), policy.ExcludedTypes...)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gzipQuality, identityQuality, ok := negotiate(r.Header.Values("Accept-Encoding"), policy.MaxHeaderBytes)
			httpx.AddVary(w.Header(), "Accept-Encoding")
			if !ok {
				httpx.SafeError(w, http.StatusNotAcceptable, "no acceptable content coding\n")
				return
			}
			buffered := newBuffer(w, policy.MaxBuffer)
			next.ServeHTTP(buffered, r)
			if buffered.spilled {
				return
			}
			if shouldCompress(r, buffered, gzipQuality, identityQuality, policy.MinimumBytes, excluded) {
				writeGzip(w, buffered, policy.Level)
				return
			}
			buffered.commitIdentity()
		})
	}, nil
}

type responseBuffer struct {
	destination http.ResponseWriter
	header      http.Header
	status      int
	buffer      bytes.Buffer
	maximum     int
	spilled     bool
}

func newBuffer(destination http.ResponseWriter, maximum int) *responseBuffer {
	return &responseBuffer{destination: destination, header: destination.Header().Clone(), maximum: maximum}
}
func (w *responseBuffer) Header() http.Header { return w.header }
func (w *responseBuffer) WriteHeader(status int) {
	if status >= 100 && status < 200 {
		copyHeader(w.destination.Header(), w.header)
		w.destination.WriteHeader(status)
		return
	}
	if w.status == 0 {
		w.status = status
	}
}
func (w *responseBuffer) Write(payload []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.spilled {
		return w.destination.Write(payload)
	}
	if w.buffer.Len()+len(payload) > w.maximum {
		w.commitIdentity()
		return w.destination.Write(payload)
	}
	return w.buffer.Write(payload)
}
func (w *responseBuffer) commitIdentity() {
	if w.spilled {
		return
	}
	w.spilled = true
	httpx.AddVary(w.header, "Accept-Encoding")
	copyHeader(w.destination.Header(), w.header)
	w.destination.WriteHeader(statusOrOK(w.status))
	_, _ = w.destination.Write(w.buffer.Bytes())
	w.buffer.Reset()
}
func shouldCompress(r *http.Request, w *responseBuffer, gzipQ, identityQ float64, minimum int, excluded []string) bool {
	status := statusOrOK(w.status)
	if gzipQ <= 0 || gzipQ < identityQ || r.Method == http.MethodHead || status < 200 || status == http.StatusNoContent || status == http.StatusNotModified || r.Header.Get("Range") != "" || w.header.Get("Content-Range") != "" || w.header.Get("Content-Encoding") != "" || strings.Contains(strings.ToLower(w.header.Get("Cache-Control")), "no-transform") || w.buffer.Len() < minimum {
		return false
	}
	mediaType, _, _ := mime.ParseMediaType(w.header.Get("Content-Type"))
	for _, excludedType := range excluded {
		if strings.EqualFold(mediaType, excludedType) {
			return false
		}
	}
	return true
}
func writeGzip(destination http.ResponseWriter, w *responseBuffer, level int) {
	header := w.header.Clone()
	header.Set("Content-Encoding", "gzip")
	httpx.AddVary(header, "Accept-Encoding")
	for _, name := range []string{"Content-Length", "ETag", "Content-MD5", "Digest"} {
		header.Del(name)
	}
	copyHeader(destination.Header(), header)
	destination.WriteHeader(statusOrOK(w.status))
	encoder, _ := gzip.NewWriterLevel(destination, level)
	_, _ = io.Copy(encoder, &w.buffer)
	_ = encoder.Close()
}
func negotiate(lines []string, maxBytes int) (gzipQ, identityQ float64, ok bool) {
	identityQ = 1
	if len(lines) == 0 {
		return 0, identityQ, true
	}
	if len(lines) == 1 && lines[0] == "" {
		return 0, identityQ, true
	}
	wildcard := -1.0
	gzipSet := false
	identitySet := false
	remaining, items := maxBytes, 0
	for _, line := range lines {
		parts, valid := httpx.SplitDelimited(line, ',', remaining, 64-items)
		if !valid {
			return 0, 0, false
		}
		remaining -= len(line)
		items += len(parts)
		for _, part := range parts {
			fields, valid := httpx.SplitDelimited(part, ';', len(part), 8)
			if !valid {
				return 0, 0, false
			}
			coding := strings.ToLower(fields[0])
			q := 1.0
			qualitySeen := false
			for _, field := range fields[1:] {
				key, value, found := strings.Cut(strings.TrimSpace(field), "=")
				if !found || !strings.EqualFold(key, "q") || qualitySeen {
					return 0, 0, false
				}
				parsed, valid := httpx.ParseQuality(value)
				if !valid {
					return 0, 0, false
				}
				q = parsed
				qualitySeen = true
			}
			switch coding {
			case "gzip":
				gzipSet = true
				if q > gzipQ {
					gzipQ = q
				}
			case "identity":
				identityQ = q
				identitySet = true
			case "*":
				wildcard = q
			}
		}
	}
	if !gzipSet && wildcard >= 0 {
		gzipQ = wildcard
	}
	if !identitySet && wildcard == 0 {
		identityQ = 0
	}
	return gzipQ, identityQ, gzipQ > 0 || identityQ > 0
}
func copyHeader(destination, source http.Header) {
	for key := range destination {
		destination.Del(key)
	}
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}
func statusOrOK(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}
