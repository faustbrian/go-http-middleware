# Behavior tables

These tables are the deployment contract for security-sensitive decisions.
Malformed means syntactically invalid, duplicated where singular, or over the
configured byte or item budget.

## Trusted proxy

| Peer and field state | Effective result |
|---|---|
| direct peer is not trusted | socket peer, request host, and TLS scheme only |
| trusted peer, valid configured syntax | first untrusted hop plus nearest host and scheme |
| trusted peer, malformed forwarding field | complete fallback to direct information |
| untrusted hop followed by spoofed earlier hops | first untrusted hop; earlier hops ignored |
| absent or empty trusted list | no forwarding field can influence the result |

`Forwarded` and `X-Forwarded-*` are mutually exclusive policy modes. Host and
prefix values are data, not redirect authorization; callers still need an
allowlist before constructing an absolute URL.

## CORS

| Request or policy | Result |
|---|---|
| no `Origin` | pass without CORS response fields |
| allowed serialized origin | echo canonical origin and add `Vary: Origin` |
| non-credentialed wildcard origin | emit `*`; no origin variation needed |
| credentials with any wildcard | constructor error |
| invalid or denied simple origin | pass application response without allow fields |
| invalid preflight origin or control-field syntax | `400`, no allow fields and no application execution |
| denied origin, method, header, or private network | `403`, no allow fields |
| accepted preflight | `204`, or application response when pass-through is set |

Origins normalize scheme, IDNA host, IPv6 brackets, and default ports. The
opaque `null` origin can be admitted by an exact configured entry, an approving
dynamic callback or non-credentialed wildcard; wildcard is not a null exclusion.
HTTP method
tokens are compared case-sensitively; header names are compared
case-insensitively. CORS is not authentication, authorization, or CSRF
protection.

## Security headers

| Policy | Result |
|---|---|
| API defaults | `nosniff`, `no-referrer`, and `DENY` |
| replace | configured value is reasserted immediately before commitment |
| preserve | downstream value wins when present |
| HSTS without acknowledgement | constructor error |
| control character or invalid field grammar | constructor error |

HSTS, CSP, and permissions policy are explicit deployment choices. No nonce,
template, session, or application security state is inferred.

## Compression

| Response or request | Result |
|---|---|
| gzip preferred and response eligible | gzip plus merged `Vary` |
| identity preferred or gzip forbidden | identity |
| every available coding forbidden | `406` before application execution |
| HEAD, range, 1xx, 204, 304, existing coding, or `no-transform` | identity |
| response below minimum or excluded media type | identity |
| response exceeds buffer while gzip is required | bounded streaming gzip |
| `101 Switching Protocols` | immediate identity commitment |

Compression removes `Content-Length`, entity tags, and digest metadata tied to
the identity representation. Custom trailers remain trailers; representation
digest trailers are removed after transformation. A panic closes an active
encoder before propagating.

## Body, deadline, timeout, and admission

| Condition | Result |
|---|---|
| known body exceeds limit | `413` before application, connection marked close |
| streaming body exceeds limit | `*http.MaxBytesError`; safe `413` if uncommitted |
| encoded or multipart body | encoded transport bytes are counted; no decoding |
| body partly read before installation | only remaining bytes; this order is unsafe |
| parent deadline is shorter | parent remains authoritative |
| context deadline expires | context is canceled; handler is not interrupted |
| buffered timeout expires | bounded safe error; late writes fail |
| timeout response overflows | safe `500`; buffered bytes are not committed |
| timeout worker capacity is full | immediate configured 5xx rejection |
| admission permit available | execute and release on every return or panic |
| bounded waiter canceled or shutdown | reject without leaking a permit |
| wait queue or wait duration exhausted | immediate `503` with optional retry hint |

Buffered timeout, body limits, and admission complement server read, write,
idle, header, connection, and shutdown limits; they do not replace them.

## Observation and privacy

| Input | Default event behavior |
|---|---|
| method or protocol | fixed known class or `OTHER` |
| route and client class | empty unless injected; bounded to 128 and 64 bytes |
| route-local recorded classification | last bounded value on this request |
| path, query, body, headers, credentials, IDs, errors, panic values | excluded |
| cancellation or panic | bounded outcome classification only |
| observer or metadata extractor panic | contained by default |

Observers run synchronously once at completion. They must impose their own
latency budget. Caller-triggered recursion is supported but remains caller
bounded; the package creates no observer worker or recursive dispatch.

## Existing implementation limits

These record current limitations, not normative protocol interpretations or
conformance success. `New`/`Described` enforce the 256-layer admission count;
`Append`/`Prepend` and `Handler` do not re-admit that count. Use `Described` or
`Concat` when enforcing the construction ceiling on extended chains.

Compression's initial 406 rejects requests with neither supported coding
acceptable. After downstream produces an ineligible response (for example below
the threshold or `no-transform`), the current implementation can emit identity
even when its negotiated quality was zero. Content admission requires any
positive matching `Accept` range; a broad positive range can override a more
specific zero-quality range and parameters do not select a representation.
Neither guard guarantees the application's actual response representation.

Buffered timeout limits payload and worker count, not header-map size, total
heap or transport write duration. Timed-out handlers retain a slot until they
return; late payload writes fail but Header still exposes a mutable map. A late
worker panic is recovered without a delivered observation. Callers must retain
cooperative cancellation and ordinary ResponseWriter/header ownership.

The transparent tracking recorder does not mark status 101 alone as committed;
it needs a payload, flush or successful hijack. This existing tracking omission
is distinct from compression's explicit 101 commitment and can affect recovery/
observation metadata for bare upgrades. It is not a net/http interpretation.

Observation bounds caller classifications by bytes, not privacy or cardinality;
truncation can split UTF-8. Injected clock panics can prevent events. Recovery
stack capture is bounded but not redacted; application observers own its privacy.
