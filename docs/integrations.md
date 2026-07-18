# Integration cookbook

## go-router

Resolve this package's server chain around the compiled router. Route-local
middleware remains visible through `go-router.NamedMiddleware`. Inject matched
route names into `observe.Policy.Route`; never use raw request paths as the
default route label.

## go-service

The current default `go-service/serverhttp` stack owns recovery, request IDs,
and body limits. Before adding an explicit chain, call
`adapter.ValidateGoService(chain, adapter.GoServiceDefaults())`. A validation
error means one implementation must be disabled or omitted. Do not install both.

## Authentication and authorization

Wrap `authhttp.NewMiddleware` from `go-authentication` with
`adapter.Named(adapter.Authentication, middleware)`. Wrap the handler returned
by `go-authorization/authhttp` at the declared authorization position. This
package never parses credentials or decides access.

## Rate limiting and idempotency

Use `go-rate-limit/ratelimithttp` and `go-idempotency/idempotencyhttp` directly.
Local `admission` protects concurrency but is not a quota. Request IDs are not
idempotency keys. Preserve each owning package's failure and replay contracts.

## Logging and telemetry

Convert `observe.Event` in an injected observer. `go-log`, `log/slog`, and
`go-telemetry` own backends, span lifecycle, exporters, and sampling. Trace
Context propagation remains in the telemetry adapter; this core neither starts
spans nor registers a global propagator.

## Recommended integration chain

`recovery -> observe -> proxy -> request ID -> CORS -> security headers ->
admission -> body/deadline -> authentication -> rate limit -> authorization ->
idempotency -> router/application -> compression`

Change order only with a documented ownership and short-circuit analysis.
