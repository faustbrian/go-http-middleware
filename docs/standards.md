# Standards scope and divergences

The package implements narrow server behavior; it does not claim full
compliance with an entire specification.

- Go 1.26.5 `net/http` handler, body, timeout, trailer, optional writer, and
  `ResponseController` contracts.
- RFC 9110 field lists, qvalues, media types, content coding selection,
  status/method body exclusions, `Vary`, and `Retry-After` syntax used here.
- RFC 9111 `no-store`, `no-transform`, and cache-key variation behavior.
- RFC 7239 bounded `Forwarded` list, quoted value, node, host, and proto parsing
  at an explicit trusted-peer boundary. Obfuscated and `unknown` client nodes
  intentionally fail closed rather than become an effective address.
- The Fetch Living Standard CORS response and preflight headers, serialized
  origins, credentials/wildcard restrictions, and cache variation. Private
  Network Access is opt-in and documented as an extension, not blanket Fetch
  compliance.
- RFC 6797 HSTS directive grammar with an additional conservative ten-year
  construction bound.

W3C Trace Context parsing and SDK/exporter lifecycle remain owned by
`go-telemetry`; this package's adapters compose owning middleware and do not
create spans or exporters.

Primary references: [Go net/http](https://pkg.go.dev/net/http),
[RFC 9110](https://www.rfc-editor.org/rfc/rfc9110.html),
[RFC 9111](https://www.rfc-editor.org/rfc/rfc9111.html),
[RFC 7239](https://www.rfc-editor.org/rfc/rfc7239.html),
[Fetch](https://fetch.spec.whatwg.org/#http-cors-protocol), and
[Trace Context](https://www.w3.org/TR/trace-context/).
