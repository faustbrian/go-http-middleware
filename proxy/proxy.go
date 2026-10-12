// Package proxy derives effective request information only across explicitly
// trusted proxy address boundaries. New derives context metadata without
// mutating the target or connecting to a proxy. RFC7239 syntax and legacy
// X-Forwarded syntax are separate policy modes; neither authorizes redirects.
// Deployments must strip attacker forwarding data at their trusted boundary.
package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/faustbrian/go-http-middleware/v2/internal/httpx"
)

// Mode selects one forwarding syntax.
type Mode uint8

const (
	// Forwarded selects the RFC 7239 Forwarded field.
	Forwarded Mode = iota
	// XForwarded selects explicitly configured X-Forwarded fields.
	XForwarded
)

// Provenance describes whether forwarding data influenced the result.
type Provenance string

const (
	// Direct means only direct connection information was used.
	Direct Provenance = "direct"
	// TrustedForwarded means trusted forwarding metadata was used.
	TrustedForwarded Provenance = "trusted_forwarded"
)

// Info contains independently derived effective values.
type Info struct {
	// ClientIP is the direct peer or trust-walk endpoint; all-trusted chains retain the leftmost supplied hop. Invalid socket metadata can yield an invalid address.
	ClientIP netip.Addr
	// Host is direct request Host or the nearest configured forwarding value; not redirect authorization.
	Host string
	// Scheme is TLS-derived http/https or the nearest admitted forwarding value.
	Scheme string
	// Prefix is an admitted X-Forwarded-Prefix value only; it never rewrites URL.Path.
	Prefix string
	// Provenance states whether trusted configured forwarding syntax influenced the result.
	Provenance Provenance
}

// Policy is copied during construction. Trusted accepts at most 256 valid
// prefixes; prefixes are stored in canonical masked form.
type Policy struct {
	// Trusted contains at most 256 copied masked peer prefixes; deployment must strip spoofed forwarding fields.
	Trusted []netip.Prefix
	// Mode selects Forwarded by default or explicit XForwarded; syntaxes are not combined.
	Mode Mode
	// MaxHops defaults to 16 and is bounded to [1,128].
	MaxHops int
	// MaxHeaderBytes defaults to 8192, maximum 1 MiB; admission applies per selected field, not all request headers.
	MaxHeaderBytes int
}

// ErrInvalidPolicy identifies invalid trusted proxy configuration.
var ErrInvalidPolicy = errors.New("proxy: invalid policy")

// ConfigError reports an invalid trusted proxy policy field.
type ConfigError struct {
	// Field names the invalid policy field without echoing its configured value.
	Field string
}

// Error formats the policy-field diagnostic; constructors use fixed field labels.
func (e *ConfigError) Error() string { return fmt.Sprintf("proxy: invalid %s", e.Field) }

// Unwrap exposes ErrInvalidPolicy for errors.Is.
func (e *ConfigError) Unwrap() error { return ErrInvalidPolicy }

type contextKey struct{}

const maximumTrustedPrefixes = 256

// New constructs trusted-proxy middleware. Malformed forwarding fields fail
// closed to direct connection information, not an HTTP refusal. Context clones
// share original request fields/body; trust lists are copied. The client walks
// inward from the peer to the first untrusted hop, or the leftmost supplied hop
// when all are trusted. Host/scheme use the nearest
// admitted forwarding metadata and still require caller redirect/authority policy.
func New(policy Policy) (func(http.Handler) http.Handler, error) {
	if policy.Mode > XForwarded {
		return nil, &ConfigError{Field: "mode"}
	}
	if policy.MaxHops == 0 {
		policy.MaxHops = 16
	}
	if policy.MaxHeaderBytes == 0 {
		policy.MaxHeaderBytes = 8192
	}
	if policy.MaxHops < 1 || policy.MaxHops > 128 {
		return nil, &ConfigError{Field: "maximum hops"}
	}
	if policy.MaxHeaderBytes < 1 || policy.MaxHeaderBytes > 1<<20 {
		return nil, &ConfigError{Field: "maximum header bytes"}
	}
	if len(policy.Trusted) > maximumTrustedPrefixes {
		return nil, &ConfigError{Field: "trusted prefixes"}
	}
	trusted := make([]netip.Prefix, len(policy.Trusted))
	for index, prefix := range policy.Trusted {
		if !prefix.IsValid() {
			return nil, &ConfigError{Field: "trusted prefix"}
		}
		trusted[index] = prefix.Masked()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info := directInfo(r)
			if isTrusted(info.ClientIP, trusted) {
				if forwarded, ok := forwardedInfo(r, info, policy.Mode, policy.MaxHops, policy.MaxHeaderBytes, trusted); ok {
					info = forwarded
				}
			}
			ctx := context.WithValue(r.Context(), contextKey{}, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}

// FromContext returns derived data, or its zero value when middleware was absent.
func FromContext(ctx context.Context) Info { value, _ := ctx.Value(contextKey{}).(Info); return value }

func directInfo(r *http.Request) Info {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	client, _ := netip.ParseAddr(strings.Trim(host, "[]"))
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return Info{ClientIP: client.Unmap(), Host: r.Host, Scheme: scheme, Provenance: Direct}
}

func forwardedInfo(r *http.Request, direct Info, mode Mode, maxHops, maxBytes int, trusted []netip.Prefix) (Info, bool) {
	if mode == Forwarded {
		return parseForwarded(r.Header.Values("Forwarded"), direct, maxHops, maxBytes, trusted)
	}
	values := r.Header.Values("X-Forwarded-For")
	if len(values) != 1 {
		return Info{}, false
	}
	parts, valid := httpx.SplitDelimited(values[0], ',', maxBytes, maxHops)
	if !valid {
		return Info{}, false
	}
	hops := make([]netip.Addr, len(parts))
	for index, part := range parts {
		address, err := netip.ParseAddr(strings.TrimSpace(part))
		if err != nil {
			return Info{}, false
		}
		hops[index] = address.Unmap()
	}
	client := selectClient(direct.ClientIP, hops, trusted)
	scheme, host, prefix := direct.Scheme, direct.Host, ""
	if value, present, valid := forwardedField(r.Header, "X-Forwarded-Proto", maxHops, maxBytes); !valid {
		return Info{}, false
	} else if present {
		value = strings.ToLower(value)
		if value != "http" && value != "https" {
			return Info{}, false
		}
		scheme = value
	}
	if value, present, valid := forwardedField(r.Header, "X-Forwarded-Host", maxHops, maxBytes); !valid {
		return Info{}, false
	} else if present {
		if !validHost(value) {
			return Info{}, false
		}
		host = value
	}
	if value, present, valid := forwardedField(r.Header, "X-Forwarded-Prefix", maxHops, maxBytes); !valid {
		return Info{}, false
	} else if present {
		if !validPrefix(value) {
			return Info{}, false
		}
		prefix = value
	}
	return Info{ClientIP: client, Host: host, Scheme: scheme, Prefix: prefix, Provenance: TrustedForwarded}, true
}

func parseForwarded(values []string, direct Info, maxHops, maxBytes int, trusted []netip.Prefix) (Info, bool) {
	if len(values) != 1 {
		return Info{}, false
	}
	elements, valid := httpx.SplitDelimited(values[0], ',', maxBytes, maxHops)
	if !valid {
		return Info{}, false
	}
	hops := make([]netip.Addr, 0, len(elements))
	result := direct
	for index, element := range elements {
		var found bool
		seen := make(map[string]bool)
		parameters, valid := httpx.SplitDelimited(element, ';', len(element), 32)
		if !valid {
			return Info{}, false
		}
		for _, parameter := range parameters {
			key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if !ok {
				return Info{}, false
			}
			key = strings.ToLower(strings.TrimSpace(key))
			if !validParameterName(key) || seen[key] {
				return Info{}, false
			}
			seen[key] = true
			value, ok = forwardedValue(value)
			if !ok {
				return Info{}, false
			}
			switch key {
			case "for":
				address, ok := parseNode(value)
				if !ok {
					return Info{}, false
				}
				hops = append(hops, address)
				found = true
			case "proto":
				if index == len(elements)-1 {
					value = strings.ToLower(value)
					if value != "http" && value != "https" {
						return Info{}, false
					}
					result.Scheme = value
				}
			case "host":
				if index == len(elements)-1 {
					if !validHost(value) {
						return Info{}, false
					}
					result.Host = value
				}
			}
		}
		if !found {
			return Info{}, false
		}
	}
	result.ClientIP = selectClient(direct.ClientIP, hops, trusted)
	result.Provenance = TrustedForwarded
	return result, true
}

func validParameterName(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if !strings.ContainsRune("!#$%&'*+-.^_`|~0123456789abcdefghijklmnopqrstuvwxyz", character) {
			return false
		}
	}
	return true
}

func parseNode(value string) (netip.Addr, bool) {
	if strings.HasPrefix(value, "_") {
		return netip.Addr{}, false
	}
	if strings.EqualFold(value, "unknown") {
		return netip.Addr{}, false
	}
	if address, err := netip.ParseAddr(strings.Trim(value, "[]")); err == nil {
		return address.Unmap(), true
	}
	addressPort, err := netip.ParseAddrPort(value)
	if err != nil {
		return netip.Addr{}, false
	}
	return addressPort.Addr().Unmap(), true
}

// Only traverse the next supplied address while the current outer peer is
// trusted. Earlier client-controlled hops cannot extend that trust chain.
func selectClient(peer netip.Addr, hops []netip.Addr, trusted []netip.Prefix) netip.Addr {
	current := peer
	for index := len(hops) - 1; index >= 0 && isTrusted(current, trusted); index-- {
		current = hops[index]
	}
	return current
}
func isTrusted(address netip.Addr, trusted []netip.Prefix) bool {
	for _, prefix := range trusted {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
func forwardedField(header http.Header, name string, maximum, maxBytes int) (string, bool, bool) {
	values := header.Values(name)
	if len(values) == 0 {
		return "", false, true
	}
	if len(values) != 1 {
		return "", true, false
	}
	parts, valid := httpx.SplitDelimited(values[0], ',', maxBytes, maximum)
	if !valid {
		return "", true, false
	}
	value := strings.TrimSpace(parts[len(parts)-1])
	if len(value) > 4096 || strings.ContainsAny(value, "\r\n\x00") {
		return "", true, false
	}
	return value, true, true
}
func validHost(value string) bool {
	if value == "" {
		return false
	}
	if len(value) > 255 {
		return false
	}
	if !httpx.ValidFieldValue(value, 255) {
		return false
	}
	if strings.ContainsAny(value, "/\\@") {
		return false
	}
	parsed, err := url.Parse("http://" + value)
	if err != nil || parsed.Host != value || parsed.Hostname() == "" || parsed.User != nil {
		return false
	}
	if port := parsed.Port(); port != "" {
		number, err := strconv.ParseUint(port, 10, 16)
		if err != nil || number == 0 {
			return false
		}
	}
	return true
}
func validPrefix(value string) bool {
	if !strings.HasPrefix(value, "/") {
		return false
	}
	if strings.HasPrefix(value, "//") {
		return false
	}
	if len(value) > 256 {
		return false
	}
	if !httpx.ValidFieldValue(value, 256) {
		return false
	}
	if strings.ContainsAny(value, "\\?#") {
		return false
	}
	_, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return path.Clean(value) == value
}

func forwardedValue(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "\"") {
		return value, value != "" && !strings.Contains(value, "\"")
	}
	if len(value) < 2 || value[len(value)-1] != '"' {
		return "", false
	}
	var result strings.Builder
	remaining := value[1 : len(value)-1]
	for len(remaining) != 0 {
		character := remaining[0]
		remaining = remaining[1:]
		if character == '\\' {
			if len(remaining) == 0 {
				return "", false
			}
			character = remaining[0]
			remaining = remaining[1:]
		}
		if character == '"' || character == '\r' || character == '\n' || character == 0 {
			return "", false
		}
		result.WriteByte(character)
	}
	return result.String(), true
}
