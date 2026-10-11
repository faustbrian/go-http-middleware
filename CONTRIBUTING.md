# Contributing

## Before Editing

1. Read [`AGENTS.md`](AGENTS.md) and the affected module's docs.
2. Run standalone Go tests in the affected module.
3. Identify owned dependencies and reverse dependants in `modules.json`.
4. Preserve unrelated work and generated/corpus provenance.

## Changes

Keep commits focused and conventional. Update every affected changelog with
the behavior and migration impact. Public API changes require compatibility
evidence and documentation. Specification behavior requires a decision record,
fixture coverage, and interoperability evidence.

New direct dependencies and dependency updates must follow the
[dependency governance policy](AGENTS.md#dependencies-and-supply-chain). Package-local
update bots are forbidden; the root policy owns every module and action update.

Repository verification is provided by the released `go-library-tools` version
and checksum in `.golib.yaml`. The Makefile is a thin local entry point;
package-specific conformance and sibling interoperability remain in
`verification/package.mk`, while the API baseline and mutation checkpoints
remain repository-owned.

Specification-backed changes must follow the
[specification governance contract](AGENTS.md#design), update
the affected stable
[decision entries](docs/specification-decisions.md), and complete the
Specification Decisions
section of the pull request template. An unresolved interpretation or stale
source pin is release-blocking; peer behavior cannot silently select policy.

Full scheduled and release mutation gates must finish with zero surviving viable
mutants. Local mutation is optional, not a universal contribution prerequisite.

Do not add package-local workflows, permanent replacements, machine-specific
paths, bypass flags, broad mutation exclusions, or aggregate quality metrics
that hide a failing package.

## Verification

Use Go 1.27.2 for development and CI; module minima remain Go 1.27.0. Root
tests need no sibling checkout or ecosystem ledger:

```bash
GOWORK=off go test ./...
```

Ordinary tests run deterministic fuzz seeds and saved corpus regressions, not
active exploration. The owned `integration/siblings` module is a separate public
dependency interoperability harness, not a requirement to clone sibling repos.
Neither module declares services or test/build tags; real HTTP tests create
their own disposable listeners.

With the checksum-pinned public verifier selected by `.golib.yaml`, use:

```bash
make inventory
make check
make ci
```

Both Make routes use bounded local checks. Base-free checks run seeds/corpora;
changed-source CI selects bounded active smoke for causally affected targets.
Ordinary CI consumes the public verifier without building it from source. One
existing admission race assertion checks canceled-wait refusal, bounded holder
cleanup and successful permit reuse, once with a two-minute timeout.
Request/body/header limits, deadline and compression cleanup, response interface
preservation, proxy trust, CORS and observation privacy changes retain their
affected boundary and real-HTTP assertions.

Short active exploration selects one target; optional local mutation selects
one package:

```bash
GOMAXPROCS=2 GOWORK=off go test -run '^$' -fuzz '^FuzzBodyLimit$' -fuzztime=2s -parallel=1 -timeout=2m ./bodylimit
golib mutation --module . --package ./admission
```

`make specification-check` validates the register offline;
`make specification-check-online` explicitly refreshes reviewed source digests.
`make check-full` retains online validation, all eight native 10000x fuzz budgets,
HTTP conformance, pinned sibling interoperability and benchmarks. For a relevant
public-consumer change, the owned harness can also run independently:

```bash
(cd integration/siblings && GOWORK=off go test ./... -count=1)
```

Preserve root and harness API/dependency contracts. Relevant release
qualification remains `golib release dry-run --module .` and the
[release preconditions](docs/release.md). Report unavailable, skipped and failed
boundaries honestly; local tests are not full assurance, hosted-CI or public
release qualification.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.
