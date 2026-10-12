// Package middleware composes explicit net/http middleware without a registry.
//
// Start with New or Described, then resolve Handler around the terminal. Layers
// are listed in request execution order and response calls unwind in reverse.
// Construction snapshots metadata, not decorator/handler closures; those remain
// caller-owned. It owns no server, router, authentication, signing, retry, replay
// ledger or shutdown. Callback cancellation is cooperative through request context.
package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Middleware decorates a borrowed net/http Handler at resolution time. Caller
// decorators own callback safety, returned handler state and nil/panic behavior;
// immutable chain metadata does not make those implementations concurrency-safe.
type Middleware func(http.Handler) http.Handler

// Predicate decides whether conditional middleware applies to a request. A
// panic propagates to the caller; cancellation is observable through r.Context.
type Predicate func(*http.Request) bool

var (
	// ErrNilHandler identifies a nil terminal handler.
	ErrNilHandler = errors.New("middleware: nil handler")
	// ErrNilMiddleware identifies a nil middleware function.
	ErrNilMiddleware = errors.New("middleware: nil middleware")
	// ErrNilPredicate identifies a nil conditional predicate.
	ErrNilPredicate = errors.New("middleware: nil predicate")
	// ErrInvalidName identifies an empty or unsafe descriptor name.
	ErrInvalidName = errors.New("middleware: invalid name")
	// ErrDuplicateName identifies repeated non-empty descriptor names.
	ErrDuplicateName = errors.New("middleware: duplicate name")
	// ErrChainTooDeep identifies a chain above the fixed construction bound.
	ErrChainTooDeep = errors.New("middleware: chain too deep")
	// ErrInvalidOrder identifies a violated named ordering requirement.
	ErrInvalidOrder = errors.New("middleware: invalid order")
)

// MaxChainDepth is the New/Described input-count ceiling. Append/Prepend and
// Handler do not re-admit that count; it is not a universal serving-depth bound.
const MaxChainDepth = 256

// ConstructionError reports invalid explicit chain configuration.
type ConstructionError struct {
	// Op identifies the failing construction stage.
	Op string
	// Index is the offending layer index; -1 denotes a non-indexed failure.
	Index int
	// Name is caller-supplied configuration and can be echoed even when invalid.
	Name string
	// Err is the wrapped cause for errors.Is/errors.As.
	Err error
}

// Error formats construction metadata and its cause; caller names are not redacted.
func (e *ConstructionError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("middleware: %s %q: %v", e.Op, e.Name, e.Err)
	}
	if e.Index >= 0 {
		return fmt.Sprintf("middleware: %s at index %d: %v", e.Op, e.Index, e.Err)
	}
	return fmt.Sprintf("middleware: %s: %v", e.Op, e.Err)
}

// Unwrap supports errors.Is and errors.As.
func (e *ConstructionError) Unwrap() error { return e.Err }

// Descriptor is an immutable, named middleware value used for inspection.
type Descriptor struct {
	name           string
	middleware     Middleware
	allowDuplicate bool
	before         []string
	after          []string
}

// DescriptorConfig declares immutable inspection and ordering metadata.
type DescriptorConfig struct {
	// Name is a nonempty label of at most 128 bytes, not a sanitized logging value.
	Name string
	// Middleware is a borrowed decorator invoked during Handler resolution.
	Middleware Middleware
	// AllowDuplicate must be true on every repeated occurrence of Name.
	AllowDuplicate bool
	// Before requires this layer before the first occurrence of each present name; copied.
	Before []string
	// After requires this layer after the last occurrence of each present name; copied.
	After []string
}

// DescriptorInfo is an independent introspection snapshot.
type DescriptorInfo struct {
	// Name is the inspection label, empty for unnamed layers.
	Name string
	// AllowDuplicate records explicit duplicate permission.
	AllowDuplicate bool
	// Before is an independent snapshot of before constraints.
	Before []string
	// After is an independent snapshot of after constraints.
	After []string
}

// Named constructs an inspectable middleware descriptor.
func Named(name string, middleware Middleware) (Descriptor, error) {
	return Describe(DescriptorConfig{Name: name, Middleware: middleware})
}

// Describe constructs middleware with explicit duplicate and order policy.
func Describe(configuration DescriptorConfig) (Descriptor, error) {
	name, middleware := configuration.Name, configuration.Middleware
	if !validName(name) {
		return Descriptor{}, &ConstructionError{Op: "name", Index: -1, Name: name, Err: ErrInvalidName}
	}
	if middleware == nil {
		return Descriptor{}, &ConstructionError{Op: "descriptor", Index: -1, Name: name, Err: ErrNilMiddleware}
	}
	if len(configuration.Before) > 64 || len(configuration.After) > 64 {
		return Descriptor{}, &ConstructionError{Op: "descriptor order", Index: -1, Name: name, Err: ErrChainTooDeep}
	}
	for _, dependency := range append(append([]string(nil), configuration.Before...), configuration.After...) {
		if !validName(dependency) || dependency == name {
			return Descriptor{}, &ConstructionError{Op: "descriptor order", Index: -1, Name: name, Err: ErrInvalidName}
		}
	}
	return Descriptor{name: name, middleware: middleware, allowDuplicate: configuration.AllowDuplicate, before: append([]string(nil), configuration.Before...), after: append([]string(nil), configuration.After...)}, nil
}

// Name returns the stable introspection name.
func (d Descriptor) Name() string { return d.name }

// Info returns copied descriptor metadata.
func (d Descriptor) Info() DescriptorInfo {
	return DescriptorInfo{Name: d.name, AllowDuplicate: d.allowDuplicate, Before: append([]string(nil), d.before...), After: append([]string(nil), d.after...)}
}

// Chain is an immutable ordered sequence of middleware descriptors.
type Chain struct {
	descriptors []Descriptor
}

// New constructs a chain from unnamed middleware.
func New(middleware ...Middleware) (Chain, error) {
	if len(middleware) > MaxChainDepth {
		return Chain{}, &ConstructionError{Op: "chain", Index: MaxChainDepth, Err: ErrChainTooDeep}
	}
	descriptors := make([]Descriptor, len(middleware))
	for index, item := range middleware {
		if item == nil {
			return Chain{}, &ConstructionError{Op: "chain", Index: index, Err: ErrNilMiddleware}
		}
		descriptors[index] = Descriptor{middleware: item}
	}
	return Chain{descriptors: descriptors}, nil
}

// Described snapshots descriptors and validates count, nil layers, names and
// ordering. Repeated nonempty names require every occurrence to permit duplicates;
// absent ordering targets impose no constraint and layers are never reordered.
func Described(descriptors ...Descriptor) (Chain, error) {
	if len(descriptors) > MaxChainDepth {
		return Chain{}, &ConstructionError{Op: "chain", Index: MaxChainDepth, Err: ErrChainTooDeep}
	}
	copied := append([]Descriptor(nil), descriptors...)
	if err := validateDescriptors(copied); err != nil {
		return Chain{}, err
	}
	return Chain{descriptors: copied}, nil
}

// Handler validates descriptors and resolves decorators from last to first.
// Nil terminal/results return ConstructionError; decorator panics propagate.
// Resolution effects are not rolled back on failure. The zero chain returns the
// terminal. Existing limitation: appended/prepended depth is not rechecked here.
// Callers own terminal/decorator lifetimes and concurrent invocation safety.
func (c Chain) Handler(terminal http.Handler) (http.Handler, error) {
	if terminal == nil {
		return nil, &ConstructionError{Op: "handler", Index: -1, Err: ErrNilHandler}
	}
	if err := validateDescriptors(c.descriptors); err != nil {
		return nil, err
	}
	resolved := terminal
	for index := len(c.descriptors) - 1; index >= 0; index-- {
		resolved = c.descriptors[index].middleware(resolved)
		if resolved == nil {
			return nil, &ConstructionError{Op: "middleware result", Index: index, Name: c.descriptors[index].name, Err: ErrNilHandler}
		}
	}
	return resolved, nil
}

// Descriptors returns an independent snapshot in request execution order.
func (c Chain) Descriptors() []Descriptor {
	return append([]Descriptor(nil), c.descriptors...)
}

// Prepend returns a new chain with descriptor first. Invalid input is retained
// for descriptor validation by Concat or Handler. No count admission is done here;
// use Described/Concat to enforce MaxChainDepth before resolving an extended chain.
func (c Chain) Prepend(descriptor Descriptor) Chain {
	result := append([]Descriptor{descriptor}, c.descriptors...)
	return Chain{descriptors: result}
}

// Append returns a metadata copy with descriptor last, without validation/count
// admission. Use Described/Concat when the New/Described count ceiling is required.
func (c Chain) Append(descriptor Descriptor) Chain {
	result := append([]Descriptor(nil), c.descriptors...)
	result = append(result, descriptor)
	return Chain{descriptors: result}
}

// Concat returns a validated chain containing c followed by suffix.
func (c Chain) Concat(suffix Chain) (Chain, error) {
	result := c.Descriptors()
	result = append(result, suffix.descriptors...)
	return Described(result...)
}

// When constructs middleware that evaluates predicate once per request. The
// wrapped middleware is resolved once during chain resolution. A nil wrapped
// result remains visible to Chain.Handler validation.
func When(predicate Predicate, middleware Middleware) (Middleware, error) {
	if predicate == nil {
		return nil, &ConstructionError{Op: "condition", Index: -1, Err: ErrNilPredicate}
	}
	if middleware == nil {
		return nil, &ConstructionError{Op: "condition", Index: -1, Err: ErrNilMiddleware}
	}
	return func(next http.Handler) http.Handler {
		wrapped := middleware(next)
		if wrapped == nil {
			return nil
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if predicate(r) {
				wrapped.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

// Ordering is all-occurrence ordering: Before targets the earliest occurrence,
// After the latest. Missing names are optional dependencies, not lookup errors.
func validateDescriptors(descriptors []Descriptor) error {
	seen := make(map[string]Descriptor, len(descriptors))
	positions := make(map[string]int, len(descriptors))
	lastPositions := make(map[string]int, len(descriptors))
	for index, descriptor := range descriptors {
		if descriptor.middleware == nil {
			return &ConstructionError{Op: "chain", Index: index, Name: descriptor.name, Err: ErrNilMiddleware}
		}
		if descriptor.name == "" {
			continue
		}
		if !validName(descriptor.name) {
			return &ConstructionError{Op: "chain", Index: index, Name: descriptor.name, Err: ErrInvalidName}
		}
		if previous, exists := seen[descriptor.name]; exists && (!previous.allowDuplicate || !descriptor.allowDuplicate) {
			return &ConstructionError{Op: "chain", Index: index, Name: descriptor.name, Err: ErrDuplicateName}
		}
		seen[descriptor.name] = descriptor
		if _, exists := positions[descriptor.name]; !exists {
			positions[descriptor.name] = index
		}
		lastPositions[descriptor.name] = index
	}
	for index, descriptor := range descriptors {
		for _, target := range descriptor.before {
			if targetIndex, exists := positions[target]; exists && index >= targetIndex {
				return &ConstructionError{Op: "order before " + target, Index: index, Name: descriptor.name, Err: ErrInvalidOrder}
			}
		}
		for _, target := range descriptor.after {
			if targetIndex, exists := lastPositions[target]; exists && index <= targetIndex {
				return &ConstructionError{Op: "order after " + target, Index: index, Name: descriptor.name, Err: ErrInvalidOrder}
			}
		}
	}
	return nil
}

func validName(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	for _, char := range name {
		if char <= ' ' || char == 0x7f || strings.ContainsRune("/\\", char) {
			return false
		}
	}
	return true
}
