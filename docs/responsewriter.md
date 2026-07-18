# ResponseWriter compatibility

| Middleware | Flusher | Hijacker | Pusher | ReaderFrom | ResponseController |
|---|---|---|---|---|---|
| chain, request ID, proxy, content, admission | unchanged | unchanged | unchanged | unchanged | unchanged |
| recovery, observe | exact underlying set | exact | exact | exact | forwarded |
| CORS, secure headers, no-store | exact underlying set | exact | exact | exact | forwarded |
| body limit | exact underlying set | exact | exact | exact | forwarded |
| buffered deadline timeout | unavailable | unavailable | unavailable | unavailable | unavailable |
| buffered compression | unavailable | unavailable | unavailable | unavailable | unavailable |

“Exact” means an optional interface is exposed only when the underlying writer
implements it. `http.ResponseController` reaches supported operations through
the wrapper's `Unwrap` behavior. Buffered policies deliberately withhold
streaming and connection takeover because replay cannot honor those contracts.

HTTP/1.1 flush, trailers, and hijacking and HTTP/2 flush and trailers are tested
on real listeners. HTTP/2 does not support hijacking. Push remains dependent on
the Go transport and may return `http.ErrNotSupported`.
