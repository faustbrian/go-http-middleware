# Performance

`BenchmarkBaseAndDeepChains`, `BenchmarkRequestID`, `BenchmarkProxyParsing`,
`BenchmarkCORSPreflight`, `BenchmarkCompression`, and
`BenchmarkAdmissionContention` report allocations and latency with `-benchmem`.
Run `make benchmark` on the target architecture before setting budgets.

Base chains allocate only what their terminal and test writer require.
Context-producing middleware necessarily allocates request context nodes.
Compression retains at most `MaxBuffer` response bytes plus bounded gzip state.
Timeout replay retains at most `MaxResponseBytes`. Proxy/CORS/content parsers
bound bytes and item counts before allocating lists.

Benchmarks are evidence, not universal service-level objectives. Record Go
version, architecture, CPU, concurrency, payload, transport, and policy with any
regression claim.
