package middleware_test

import (
	"net/http"
	"testing"

	middleware "github.com/faustbrian/go-http-middleware/v2"
)

// FuzzDescriptorNames accepted arbitrary descriptor names must remain resolvable through chain construction; rejection is allowed.
func FuzzDescriptorNames(f *testing.F) {
	f.Add("request-id")
	f.Add("bad name")
	f.Fuzz(func(t *testing.T, name string) {
		descriptor, err := middleware.Named(name, passthrough)
		if err != nil {
			return
		}
		chain, err := middleware.Described(descriptor)
		if err != nil {
			t.Fatalf("valid descriptor rejected: %v", err)
		}
		if _, err = chain.Handler(http.NotFoundHandler()); err != nil {
			t.Fatalf("valid chain rejected: %v", err)
		}
	})
}
