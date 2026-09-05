# Verification evidence

The release gate is reproducible from a clean checkout with `make ci`.
No test requires a running database, proxy, cache, exporter, or other
production service.

| Evidence | Blocking command and acceptance |
|---|---|
| behavior and HTTP integration | shared tests, race, and typed conformance gates; all pass |
| sibling interoperability | typed interoperability operation; pinned router and service contracts pass |
| race safety | shared race gate; no race report |
| meaningful coverage | shared coverage gate; every production package is 100.0% |
| fuzz smoke | declared typed fuzz operations; every target completes without failure |
| mutation | shared mutation gate; 59/59 curated security decisions killed |
| leaks | shared tests include the middleware leak contract; no middleware-owned goroutine |
| standards policies | shared tests and typed conformance operation; proxy, CORS, content, compression, and headers pass |
| response capabilities | real HTTP/1.1 and HTTP/2 plus nested interface matrix pass |
| privacy | observation tests exclude payload data and bound all labels |
| performance | shared benchmark gate; latency and allocations are reported with parameters |
| dependencies | shared vulnerability, license, SBOM, and module checks pass |
| architecture | shared safety and dependency checks; no forbidden runtime mechanism or sibling dependency |
| static quality | shared vet, lint, Staticcheck, docs, API, tidy, and format checks pass |
| advisory nil analysis | shared NilAway analysis; visible but intentionally non-blocking |

Fuzzing covers descriptor names, request IDs, request body limits, forwarding
fields, CORS origins and preflights, media negotiation, content coding, and
configured security headers. The curated mutation
set targets composition depth, duplicates, conditions, short circuits, trust
selection, CORS decisions, coding and media negotiation, limits, cancellation,
timeouts, status commitment, recovery, and observation privacy. Each mutant
runs in an isolated copy with a five-second test timeout.

Benchmarks cover empty and deep chains, request IDs, proxy parsing, CORS
preflights, compression, and contended admission. Results are machine-specific;
the release record must preserve Go version, OS, architecture, CPU, payload,
concurrency, and benchmark duration with any regression claim.

## Historical local release run

On 2026-07-18, `make ci` passed with Go 1.26.5 on Darwin arm64,
Apple M4 Max. It reported 100.0% production statement coverage, 59/59 killed
mutants, eight two-second fuzz targets, no race, leak, vulnerability, lint,
Staticcheck, API, documentation, architecture, or NilAway finding, and green
real HTTP plus pinned sibling integration suites.

This retained snapshot is machine-specific historical evidence, not the
current repository verification status.

| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| empty chain | 389.8 | 1,056 | 11 |
| 128-layer chain | 383.2 | 1,056 | 11 |
| request ID | 577.4 | 1,521 | 18 |
| trusted proxy parse | 989.5 | 1,945 | 18 |
| CORS preflight | 1,854 | 1,968 | 36 |
| gzip response | 42,645 | 817,428 | 58 |
| contended admission | 536.8 | 1,062 | 11 |

These 100 ms benchmark samples are evidence for this machine, not portable
service-level objectives. The enforced observation ceiling is separately
machine-independent.
