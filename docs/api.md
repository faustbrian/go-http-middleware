# API reference

The authoritative symbol list is Go documentation plus `api/baseline.txt`.
This page explains the contracts that types alone cannot express.

`Middleware` is `func(http.Handler) http.Handler`. `New` accepts unnamed
middleware. `Describe` adds a stable name, explicit duplicate permission, and
`Before`/`After` constraints. `Described` validates names, duplicates, order,
nil values, and the 256-layer bound. `Handler` also rejects nil terminals and
nil results.

Constructors return typed configuration errors supporting `errors.Is` and
`errors.As`. Runtime handlers cannot return Go errors, so short circuits use
HTTP responses and injected observers. Policies are copied during construction;
function fields and state sources must be safe for concurrent calls.

`When` evaluates its predicate once per request. Predicate panic propagates.
Cancellation is available through the request context; the condition does not
invent an error channel or recover policy.

Each subpackage's exported policy documents its defaults and bounds in Go doc.
