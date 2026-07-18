# Security deployment

## Trusted proxies

List only direct proxy address ranges you operate. Ensure each trusted proxy
removes client-supplied forwarding fields before appending its own. Choose
either RFC 7239 `Forwarded` or the explicitly configured `X-Forwarded-*` mode.
Malformed or oversized fields fail closed to direct connection data. Never use
effective host or scheme without an allowlist when building redirects.

## CORS

List exact origins whenever credentials are enabled. Wildcard method, header,
exposure, and origin configurations are rejected with credentials. CORS only
controls browser response visibility; it is not authentication, authorization,
or CSRF protection.

## HSTS and headers

HSTS requires `AcknowledgeHSTS`. Confirm HTTPS works for every covered host
before enabling a long max age or subdomains. CSP is opt-in because this package
cannot infer scripts, templates, nonces, or application assets.

## Compression

Set `Cache-Control: no-transform` for secrets reflected near attacker-controlled
input. Buffered compression skips ranges, existing encodings, no-body statuses,
HEAD, and small responses. It removes representation-specific length, digest,
and entity-tag fields when coding changes.

## IDs, bodies, and timeouts

Inbound IDs are untrusted by default and never authorization evidence. Body
limits apply to encoded transport bytes before decoding. Context deadlines do
not interrupt code that ignores context. Buffered handler timeouts cap retained
output and intentionally reject streaming capabilities.
