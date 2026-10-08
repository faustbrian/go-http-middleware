# Dependency audit

The runtime graph uses `x/net` v0.59.0 and `x/text` v0.42.0 to preserve CORS
origin identity through Unicode normalization. Testify v1.12.1 is a test-only
refresh reached through `goleak` tests, not an IDNA runtime requirement.
Graph-only modules are not compiled into this module; their selection belongs
to an upstream module graph.

| Module group | Scope and necessity | License | Cost and maintenance decision |
|---|---|---|---|
| `github.com/felixge/httpsnoop` v1.1.0 | runtime; exact optional writer interfaces | MIT | retained; replacing it risks interface lies |
| `golang.org/x/net` v0.59.0 | runtime; IDNA Lookup profile | BSD-3-Clause | retained for Fetch origin serialization |
| `golang.org/x/text` v0.42.0 | runtime transitive; IDNA Unicode tables | BSD-3-Clause | unavoidable through `x/net/idna` |
| `go.uber.org/goleak` v1.3.0 | test; goroutine leak gate | Apache-2.0 | retained outside production builds |
| `testify` v1.12.1, including its bundled Spew and Difflib sources, and `go.yaml.in/yaml/v3` v3.0.5 | transitive tests of `goleak` | MIT; bundled ISC and BSD-3-Clause; YAML MIT and Apache-2.0 | no production packages or API surface |
| `x/crypto`, `x/mod`, `x/sync`, `x/sys`, `x/term`, `x/tools` | graph-only `x/net` modules | BSD-3-Clause | `go mod why` reports no needed package |
| `kr/pretty`, `check.v1` | graph-only test modules | MIT, BSD-2-Clause | `go mod why` reports no needed package |

The production build has three external module dependencies and uses no cgo,
unsafe package, linkname, runtime patch, plugin loader, or hidden exporter.
Command-line quality tools are version-pinned in the Makefile and are invoked
with `go run`; they are not library dependencies.

[`integration/siblings`](../integration/siblings/README.md) is a separate,
non-releasable test module. It pins the compatible Golib module versions used
by the interoperability checks while keeping them out of the production module
graph.
