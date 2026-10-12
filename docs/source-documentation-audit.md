# Source documentation audit

## Pre-edit scope and checks

Base `09a9adcd30efd8612190ce59e93d1f4b31961251`; comments/docs-only Tier A in
an isolated detached main-source worktree. The dirty release checkout stays
untouched. Root `/v2` module has 16 catalog packages; `integration/siblings` is
a separate test-only published-dependency module with three test files, no
production package or buildable GoDoc entrypoint. All 77 tracked Go files and
both modules are included, not only exported symbols.

Reproduce with `git ls-files '*.go'`, both go.mod files, catalogs and shared
`source-docs inventory --json` / `source-docs check --json`. Initial checker
finds 137 exported-member/method omissions; counts locate gaps, not useful
contract proof. No generated Go, generator directives, build/platform variants
or unfinished debt markers exist. Eight fuzz targets, six benchmarks, two
runnable root Examples, real-HTTP response/interface tests and nested provider
composition tests are inventoried; their unchanged runtime claims are not
requalified by this prose pass.

## Complete tracked Go inventory

- `adapter`: `adapter.go`, `adapter_test.go`, `edge_test.go`.
- `admission`: `admission.go`, `admission_test.go`, `edge_test.go`, `stress_test.go`.
- `.`: `benchmark_test.go`, `chain.go`, `chain_edge_test.go`, `chain_test.go`, `composition_audit_test.go`, `example_test.go`, `fuzz_test.go`, `http_integration_test.go`, `integration_profiles_test.go`, `leak_test.go`.
- `bodylimit`: `audit_test.go`, `bodylimit.go`, `bodylimit_test.go`, `edge_test.go`, `fuzz_test.go`.
- `compress`: `compress.go`, `compress_test.go`, `edge_test.go`, `fuzz_test.go`, `mutation_boundaries_test.go`.
- `content`: `content.go`, `content_test.go`, `edge_test.go`, `fuzz_test.go`, `mutation_boundaries_test.go`.
- `cors`: `cors.go`, `cors_test.go`, `edge_test.go`, `fuzz_test.go`, `mutation_boundaries_test.go`, `normalization_test.go`.
- `deadline`: `deadline.go`, `deadline_test.go`, `edge_test.go`, `mutation_boundaries_test.go`, `timeout.go`.
- `integration/siblings`: `authentication_v2_test.go`, `platform_test.go`, `siblings_test.go`.
- `internal/httpx`: `header.go`, `header_test.go`, `mutation_boundaries_test.go`, `response.go`, `response_test.go`.
- `middlewaretest`: `interfaces_test.go`, `middlewaretest.go`, `middlewaretest_test.go`.
- `observe`: `audit_test.go`, `edge_test.go`, `observe.go`, `observe_test.go`.
- `proxy`: `edge_test.go`, `fuzz_test.go`, `mutation_boundaries_test.go`, `proxy.go`, `proxy_test.go`.
- `recovery`: `edge_test.go`, `recovery.go`, `recovery_test.go`.
- `requestid`: `edge_test.go`, `fuzz_test.go`, `requestid.go`, `requestid_test.go`.
- `responsepolicy`: `edge_test.go`, `responsepolicy.go`, `responsepolicy_test.go`.
- `secureheader`: `edge_test.go`, `fuzz_test.go`, `secureheader.go`, `secureheader_test.go`.

## Contract seams and useful missing documentation

- Root explicit reverse decorator resolution/request-forward order, duplicate
  named metadata, first/last position constraints, detached metadata snapshots,
  borrowed decorator/predicate closures and startup versus serving validation.
  Chain immutability does not make caller closures safe; callbacks can block or
  panic. Root owns no router, server, authentication, signing, replay ledger,
  retry, domain idempotency or provider shutdown.
- Admission channels share state across all uses of a returned factory; bounded
  wait timers, cancel-after-permit arbitration, deferred permit release, caller
  shutdown ownership without closing library channels or draining active work.
- Body limits mutate request Body/Close, count only visible unread transport
  bytes, preserve application overflow errors and cannot rewrite committed output.
  Context deadline clones retain the body; buffered timeout has distinct writer,
  worker, output and late-panic/header ownership semantics, not forced shutdown.
- Tracking/commitment, informational status and hijack rules; transparent wrappers
  preserve httpsnoop capabilities while timeout/compression intentionally hide
  them. Header policy reassertion is not transport delivery or arbitrary safety.
- Proxy peer trust/right-to-left client selection and nearest host/scheme;
  configured syntax limits do not validate redirect authorization. CORS canonical
  IDNA origin policy, preflight-only short circuits, borrowed dynamic callback,
  credential/wildcard restrictions and variation. Security headers validate
  selected grammar, not CSP/Permissions-Policy semantics or deployment safety.
- Content positive-range existence is not negotiated representation selection;
  compression per-response commitment, trailers, identity fallback, buffer limits
  versus total memory, ignored final I/O errors and lack of body decompression.
- Recovery same-goroutine scope/ErrAbortHandler and bounded stack sensitivity;
  observation retains synchronous callbacks, trusted caller classifications,
  byte truncation, panic precedence and injected-clock assumptions. IDs are
  correlation metadata, not authenticated identity or replay protection.
- Test fixtures and nested public providers exercise exact ownership/routing/auth
  integration at pinned releases, not the new worktree unless consumed version
  changes. Maintain their literals, dependencies and test behavior unchanged.

## Existing limitations versus unanswered decisions

Source reveals Append/Prepend defer descriptor validation but do not recheck
MaxChainDepth in Handler; New/Described alone enforce that construction count.
Compression can emit identity after an eligibility veto even if identity q=0;
content acceptance uses any positive match rather than most-specific exclusion.
Transparent tracking also does not mark bare 101 as committed until payload,
flush or hijack. These are existing limits, not invented specification policies.
CORS wildcard also permits opaque null origins; malformed/denied ordinary
requests reach downstream without granting browser access. Current maintained
prose contradicting those facts requires narrow clarification.

Unknown decisions: whether deferred chain construction should re-admit depth,
whether compression should refuse ineligible identity-forbidden output, and
whether media-range specificity should override broad positive acceptance.
Buffered timeouts cannot stop handlers, bound header maps or guarantee complete
response delivery. Borrowed callbacks own privacy, latency and concurrent use.
No behavioral repair or hostile-input campaign is authorized here.

## Authority and selected verification

Preserve 15 typed decisions, append-only history, monitoring pins, fixtures and
both dependency graphs. Reviewed authorities retain their recorded editions:
Go1.26.6 net/http/context, RFC9110/9111/7239/6797/7034 and pinned living Fetch/
Referrer Policy. Producer toolchain Go 1.27.2 is distinct from historical authority
pins. Existing fixtures/decisions define narrow behavior, not blanket compliance.

Explicitly granted serialized Go 1.27.2 session: formatting; all 77 comment-free
AST/signature/significant-token/literal/directive comparison; all 16 root GoDocs;
ordered native full go-doc baseline regeneration twice (normalize trailing
newlines to one), root Example selector and appropriate vet only if required.
No nested published-consumer suite, broad lifecycle/race/fuzz/bench/conformance
campaign. Native baseline mode is go-doc `api/baseline.txt`, not apidiff.
Static source, offline specification, local links/markers and pinned final
added/staged-content secrets plus one ordinary complete Tier A review precede
explicit-path Intent/ACK safe-wrapper commit. All three owned Go caches/helpers
clean immediately on success/failure; parent owns publication/CI/ledger.

## Results

- Go 1.27.2 formatting passes for all 77 tracked files. Exact executable AST/
  signatures, literals and directives are preserved across both modules.
  Raw tokens differ in 14 files solely through optional implicit semicolons before
  closing braces from idiomatic multiline field comments; normalization removes
  only those grammar-optional tokens. Significant-token comparison passes.
  All 320 comment groups parsed/rendered; grouped declarations remain grouped.
- All 16 configured root packages rendered; native full go-doc baseline generated
  twice stably in configured order, with final trailing newlines normalized to
  one. Final affected deadline/httpx/requestid package prose was rerendered twice;
  unchanged 13 outputs were reused. Nested test-only module has no production
  GoDoc output; its three files remain included in parser/signature proof.
- Existing root Examples pass with `go test ./... -run '^Example' -count=1`;
  other root packages compile with no selected tests. No nested consumer suite,
  race, active fuzz, benchmark, conformance, live service or transport campaign.
- Four disposable imported-worktree-source observations confirm depth 257 after
  Append, identity despite q=0 after small-response eligibility veto, positive
  wildcard media admission despite specific q=0, and wildcard null-origin
  admission. Bare 101 tracking remains source-verified, not transport-probed.
  No defect was repaired or reclassified as intended protocol behavior.
- Shared source checker has zero findings; offline 15 decisions validate and 13
  protected specification/dependency/catalog records remain byte-identical.
  Native baseline prose is refreshed, not frozen authority/history. Applicable
  local links, markers and pinned final added/staged secrets accompany handoff.
- Final truth review corrected all-trusted proxy endpoint and dynamic null-origin
  wording: affected 12 source files retain AST/signatures/tokens/literals/directives
  and render 63 comments; CORS/proxy outputs regenerate twice stably. Unchanged
  14 native outputs and the original Example/probe results were reused.
- Every bounded Go run removed exact owned GOCACHE/GOMODCACHE/GOTMPDIR on exit;
  imported-source probe removed. Parser/render/inventory resources removed after
  capturing this evidence. Parent-owned CLI/scanner are retained; parent owns CI.

## Objective Enforcement

The checksum-pinned public Tools v2.4.0 verifier now supplies
`make source-docs-check` as a prerequisite of the existing local and CI Make
routes. Ordinary shared CI selects strict source-documentation checking and
requires its result. Native prerequisites, operation budgets, security and
release controls are retained. Hosted exact-head evidence is recorded by the
coordination ledger; configuration alone does not establish green delivery.
