# Troubleshooting

## `invalid order`

Inspect `Chain.Descriptors()` and the descriptor's `Before`/`After` metadata.
The list is request order, not response unwind order.

## CORS header absent

Check the serialized origin including default port, credentials/wildcard
combination, method/header allowlists, field count, and byte limit. Denied actual
requests still run the application but omit allow headers; denied preflights
short circuit.

## Response is not compressed

Check `Accept-Encoding`, qvalues, identity preference, status, method, range,
existing `Content-Encoding`, `Cache-Control: no-transform`, content type,
minimum size, and maximum buffer spill.

## Body overflow returned success

The application committed a response after ignoring `*http.MaxBytesError`.
Handle body read errors before writing. The middleware can supply 413 only while
it still owns an uncommitted response and closes reuse after overflow.

## Writer interface missing

Use `middlewaretest.CapabilitiesOf` at the failing layer. Buffered timeout and
compression deliberately remove optional capabilities; other wrappers preserve
the underlying set.
