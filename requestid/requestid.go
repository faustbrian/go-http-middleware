// Package requestid propagates bounded request and correlation identifiers.
// Start with New and read typed context metadata through FromContext. Policies
// are copied, generator closures borrowed; concurrent requests require safe
// generators. No signing, deduplication, server or worker lifecycle is owned.
// Identifiers are metadata and are never authorization evidence.
package requestid

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/faustbrian/go-http-middleware/v2/internal/httpx"
)

// Kind distinguishes identifier purposes.
type Kind string

const (
	// Request identifies a request identifier.
	Request Kind = "request"
	// Correlation identifies a cross-request correlation identifier.
	Correlation Kind = "correlation"
	// Operation identifies an application operation identifier.
	Operation Kind = "operation"
)

// InvalidPolicy controls trusted invalid inbound identifiers.
type InvalidPolicy uint8

const (
	// ReplaceInvalid replaces invalid trusted inbound identifiers.
	ReplaceInvalid InvalidPolicy = iota
	// RejectInvalid rejects invalid trusted inbound identifiers.
	RejectInvalid
)

// ErrInvalidPolicy identifies invalid identifier policy configuration.
var ErrInvalidPolicy = errors.New("requestid: invalid policy")

// ConfigError reports an invalid identifier policy field.
type ConfigError struct {
	// Field names the invalid policy field without echoing its configured value.
	Field string
}

// Error formats the policy-field diagnostic; constructors use fixed field labels.
func (e *ConfigError) Error() string { return fmt.Sprintf("requestid: invalid %s", e.Field) }

// Unwrap exposes ErrInvalidPolicy for errors.Is.
func (e *ConfigError) Unwrap() error { return ErrInvalidPolicy }

// Policy values are copied by New; Generator closures remain borrowed and must
// support concurrent calls when the handler is shared.
type Policy struct {
	// Kind is Request by default; Correlation and Operation use independent context keys.
	Kind Kind
	// Header names the inbound field; defaults to X-Correlation-ID for Correlation, X-Request-ID otherwise.
	Header string
	// ResponseHeader defaults to Header and is set before downstream execution.
	ResponseHeader string
	// TrustInbound allows one admitted inbound value; false generates instead of trusting supplied IDs.
	TrustInbound bool
	// Invalid chooses replacement or rejection only for supplied trusted invalid values; zero replaces.
	Invalid InvalidPolicy
	// MaxLength caps identifier bytes in [1,1024]; zero defaults to 128.
	MaxLength int
	// Generator is borrowed synchronous caller code; nil uses crypto/rand.Text. Errors become fixed 500 responses, panics propagate.
	Generator func() (string, error)
}

type contextKey struct{ kind Kind }

// New constructs identifier middleware. Defaults are X-Request-ID, 128 bytes,
// untrusted inbound values, replacement on invalid input, and crypto/rand.Text.
// Valid trusted IDs are reused without deduplication; missing IDs still generate
// under RejectInvalid. Request context is cloned without rewriting inbound headers;
// the selected response header is set before downstream can overwrite it.
// Empty, non-ASCII, surrounding-space, control-character, and oversized
// values are invalid.
func New(policy Policy) (func(http.Handler) http.Handler, error) {
	if policy.Kind == "" {
		policy.Kind = Request
	}
	if policy.Kind != Request && policy.Kind != Correlation && policy.Kind != Operation {
		return nil, &ConfigError{Field: "kind"}
	}
	if policy.Header == "" {
		if policy.Kind == Correlation {
			policy.Header = "X-Correlation-ID"
		} else {
			policy.Header = "X-Request-ID"
		}
	}
	if policy.ResponseHeader == "" {
		policy.ResponseHeader = policy.Header
	}
	if !validHeaderName(policy.Header) || !validHeaderName(policy.ResponseHeader) {
		return nil, &ConfigError{Field: "header"}
	}
	if policy.MaxLength == 0 {
		policy.MaxLength = 128
	}
	if policy.MaxLength < 1 || policy.MaxLength > 1024 {
		return nil, &ConfigError{Field: "max length"}
	}
	if policy.Invalid > RejectInvalid {
		return nil, &ConfigError{Field: "invalid policy"}
	}
	if policy.Generator == nil {
		policy.Generator = randomIdentifier
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identifier := ""
			values := r.Header.Values(policy.Header)
			if policy.TrustInbound && len(values) == 1 && validIdentifier(values[0], policy.MaxLength) {
				identifier = values[0]
			} else if policy.TrustInbound && len(values) > 0 && policy.Invalid == RejectInvalid {
				httpx.SafeError(w, http.StatusBadRequest, "invalid request identifier\n")
				return
			}
			if identifier == "" {
				generated, err := policy.Generator()
				if err != nil || !validIdentifier(generated, policy.MaxLength) {
					httpx.SafeError(w, http.StatusInternalServerError, "internal server error\n")
					return
				}
				identifier = generated
			}
			w.Header().Set(policy.ResponseHeader, identifier)
			ctx := context.WithValue(r.Context(), contextKey{kind: policy.Kind}, identifier)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}

// FromContext returns an identifier under the exact Kind key. Missing/unknown
// kinds return false; nil context is not accepted and panics. IDs are metadata,
// not proof of identity, uniqueness, exactly-once work or replay refusal.
func FromContext(ctx context.Context, kind Kind) (string, bool) {
	value, ok := ctx.Value(contextKey{kind: kind}).(string)
	return value, ok
}

func randomIdentifier() (string, error) {
	return rand.Text(), nil
}

func validIdentifier(value string, maximum int) bool {
	if value == "" || len(value) > maximum || strings.TrimSpace(value) != value {
		return false
	}
	for _, char := range value {
		if char < 0x21 || char > 0x7e {
			return false
		}
	}
	return true
}

func validHeaderName(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if !strings.ContainsRune("!#$%&'*+-.^_`|~0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", char) {
			return false
		}
	}
	return true
}
