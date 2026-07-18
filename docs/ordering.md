# Ordering reference

For `[A, B, C]`, request execution is `A -> B -> C -> handler`; response unwind
is `handler -> C -> B -> A`. A short circuit prevents every inner layer from
running. An outer observation layer sees inner short circuits.

| Layer | Recommended position | Reason |
|---|---|---|
| recovery | outermost | contains every inner application panic |
| observation | inside recovery | recovery owns panic response; observation sees completion |
| trusted proxy | before client-based owning policy | establishes bounded effective data |
| request ID | before logs and telemetry | gives inner observers context metadata |
| CORS/security headers | outside application errors | applies headers at commitment |
| admission | before expensive work | rejects overload early |
| body limit/deadline | before decoders and application | owns read/time boundary |
| authentication/authorization | owning package order | this module does not decide access |
| idempotency/rate limit | owning package order | this module does not duplicate state |
| compression | outside representation handler | transforms final eligible representation |

Named descriptors make security-sensitive constraints inspectable. Duplicate
names fail unless every duplicate explicitly permits repetition. Use
`adapter.ValidateGoService` before combining a chain with `go-service` defaults.
