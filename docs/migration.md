# Migration guide

## Version 2

Use Go 1.27.0 or newer and require
`github.com/faustbrian/go-http-middleware/v2 v2.0.0` once that version is
published. Update every root and subpackage import to include `/v2`, including
`adapter` and `middlewaretest`. Package names and exported symbols are
otherwise unchanged. Do not create a version-specific source directory.

Malformed CORS preflights fail closed before application routing, and the
configured content-negotiation byte limit applies to Content-Type before
parsing. Honor these rejections rather than relying on the old fallthrough.
Compression honors no-transform on every Cache-Control field line, and panic
stack capture uses the configured bounded buffer.

The nested `integration/siblings` module is a non-releasable harness with its
own identity. It deliberately retains the published v1 dependency until v2
is public; it is not evidence that v2 consumers have migrated. The same
publication boundary applies to sibling-owned consumer harnesses. Source
preparation and the dated changelog do not establish tag or release publication.

## Ad hoc net/http

Inventory every existing wrapper, its request order, response order, short
circuit, and optional writer behavior. Replace one concern at a time with an
explicit constructor and characterization test. Remove the old layer before
enabling the new named descriptor.

## Laravel middleware

Translate the visible route/kernel order into `Chain` descriptors. There are no
aliases, groups, container resolution, request DTOs, sessions, CSRF views,
guards, controllers, facades, or exception renderers here. Keep validation,
authentication, authorization, and protocol errors in their owning Go package.

## Common Go middleware stacks

Do not copy an implicit global stack. Construct policies near server setup,
inspect descriptors, resolve once, and pass the resulting `http.Handler` to the
server. Test `Flusher`/`Hijacker` behavior before replacing wrappers used by
streaming or upgrades. Buffered compression and timeout middleware require a
non-streaming handler. Size `TimeoutPolicy.MaxConcurrent` for the maximum
number of downstream executions that may overlap, including handlers that
ignore cancellation after their response timeout.

## service overlap

Choose whether `service` or this package owns recovery, request IDs, and body
limits. Current `service` defaults own them; use the adapter validator and
omit duplicates until the server is configured to delegate ownership.
