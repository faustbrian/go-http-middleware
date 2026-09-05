# Changelog

This project follows Semantic Versioning and Keep a Changelog.

## Unreleased

### Changed

- Publish the machine-auditable [specification decision register](docs/specification-decisions.md)
  with authority monitoring, conformance bindings, and durable decision
  history for the HTTP middleware policy surface.

Current specification decision content:

- HTTPMIDDLEWARE-DEC-001 sha256:ac7f1a971688a278debc11526319e41b5a9fde65029e5b8116af94031774f7cf
- HTTPMIDDLEWARE-DEC-002 sha256:32f235a5b6904cc389028f57d64710f05dd8d16bef4682b801dfa9df18801c98
- HTTPMIDDLEWARE-DEC-003 sha256:5ca1bec6225929c7a25aff601ed17148960bac465df08c83a1364caaeb77abf1
- HTTPMIDDLEWARE-DEC-004 sha256:1061435c5c0169974df8f7a816a4e93816e09430c71c96d9bb4069917ece6907
- HTTPMIDDLEWARE-DEC-005 sha256:116bfc4a2fd0cf94a9a51251a6b86e4270629e82489a6a886369489e9664bb91
- HTTPMIDDLEWARE-DEC-006 sha256:b9e73bafaf90c1f517c1f45473f4eac1ec3f622633078ed3714622b07c4fd6ec
- HTTPMIDDLEWARE-DEC-007 sha256:5d0330131bfae527456e203b65aadbd9559545b85e5b14cf43160ddaab7ccee8
- HTTPMIDDLEWARE-DEC-008 sha256:42307e67ed4072251057a97cfd61d7d453fb163bfdf6a7788be9ca38e3d0768b
- HTTPMIDDLEWARE-DEC-009 sha256:e66a0435f1e69599c8c58147fe405093e506a8570f9f6f4475d0b996ac89415d
- HTTPMIDDLEWARE-DEC-010 sha256:e6c3cb35526dd48a121e5180f3e2a30d67adb5bcae9e0e6861de2aeb5fa2d786
- HTTPMIDDLEWARE-DEC-011 sha256:0450178874719bd880e705596c8f2bfcc64cc8a725cdc8098592d31ad9be93e9
- HTTPMIDDLEWARE-DEC-012 sha256:477e63e3c48ef6087dfb88d593bb0be02be5cc556940f8edf9b2e3c76082098c
- HTTPMIDDLEWARE-DEC-013 sha256:c56c51381ab6f0bb613c2a31f3e0bb00ee0bceeaf4c7944a5a014ac3e1084da3
- HTTPMIDDLEWARE-DEC-014 sha256:5b857207fd48f586341935eebc2aeac2c8c7e930f280df330d72d4f83d3a5717
- HTTPMIDDLEWARE-DEC-015 sha256:100f655f802a18504974028f22573999f0e4157485641054fe456f4c0ed9d4d7

- Adopt the checksum-verified `go-library-tools` v1.2.0 CLI and immutable
  shared workflow so local and hosted gates enforce specification governance
  while preserving the API baseline, mutation checkpoints, HTTP conformance
  tests, sibling integration harness, and package fixtures.
- Adopt the checksum-verified `go-library-tools` v1.3.0 CLI, schema-v2 cohesion
  metadata, repository-local validation target, and immutable shared workflow.
- Adopt the checksum-verified `go-library-tools` v1.4.0 CLI and immutable
  shared workflow so authority monitoring uses the stabilized request profile
  and public-first module resolution.
- Reconcile the sibling integration harness with the immutable public SumDB
  identities for its Golib v1.0.0 dependencies.

### Documentation

- Document the non-releasable sibling interoperability harness and distinguish
  its historical verification snapshot from current repository status.
- Add immutable v1.4.0 ecosystem and service-edge family navigation.
- Record the behavior-neutral reviews of Go 1.26.7 through 1.26.8, RFC 9110
  Erratum 9162, current WHATWG Fetch and URL sources, and W3C Referrer Policy;
  the selected middleware behavior and conformance bindings remain unchanged.

- Remove completed implementation plans from the release tree and retain
  package-owned documentation as the maintained reference.
- Link the public module to the versioned Golib ecosystem design language and
  package-selection guidance.

## 1.0.0 - 2026-08-25

### Changed

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Documentation

- Replace obsolete standalone-repository links and workflow claims with
  monorepo-canonical targets and current release guidance.

### Added

- Explicit immutable middleware chains and named order descriptors.
- Bounded request ID, recovery, body limit, deadline, trusted proxy, CORS,
  security header, compression, observation, content, admission, and response
  policy packages.
- HTTP/1.1 and HTTP/2 integration fixtures, fuzzing, mutation checks,
  benchmarks, ownership adapters, and release automation.
- A bounded request-scoped route recorder for routers that clone requests.
- A pinned specification decision register and focused conformance gate for
  HTTP, Fetch, URL, forwarding, HSTS, and Go runtime behavior.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-http-middleware` identity while preserving its documented API and behavior.
- Link the conformance source matrix directly to the canonical specification
  decision register.
- Regenerated the exported API baseline with the pinned Go documentation
  formatter without changing the public contract.
- Bound trusted-prefix and configured media-policy collections, handler
  deadlines, admission waits, and observation method/protocol cardinality.
- Bound context-ignoring buffered-timeout executions with an explicit
  per-middleware concurrency limit.
- Compare CORS preflight methods with HTTP's case-sensitive method semantics.

### Fixed

- Prove that requests admitted from free capacity bypass the bounded waiter
  queue instead of inheriting its configured delay.
- Give the real-listener informational-timeout assertion enough scheduling
  budget to remain deterministic under parallel CI load.
- Make admission waiter-bound verification deterministic under heavily
  parallel coverage execution.
- Preserve acceptable gzip coding after buffer spill and close streaming
  encoders during panic unwind.
- Reject control characters in identifiers, malformed media wildcards and
  parameters, and canceled requests before admission.
- Reject nil conditional results and ordering constraints placed between
  duplicated target layers.
- Preserve informational responses through buffered timeout, commit protocol
  switches through compression, and reject invalid status codes.
- Reject malformed wildcard CORS methods, invalid origin ports, duplicate
  Forwarded parameters, and non-ASCII identifiers.
- Extract route and client-class metadata after downstream completion and
  contain metadata-extractor panics.
- Reject duplicate content types and malformed or oversized Accept tails even
  when an earlier media range matches.
- Preserve response trailers through compression while removing stale digest,
  length, and entity-tag metadata for the identity representation.
- Preserve implicit identity encoding preference and accept bounded Unicode
  origins whose IDNA serialization is valid.
