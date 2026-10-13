# Release process

[Documentation map](index.md) · [Repository README](../README.md)
[Next: API reference](api.md)

1. Run `make check-full` on a clean tree with the declared Go version.
2. Confirm architecture restrictions, 100.0% production coverage, all declared
   mutants killed, fuzz smoke, race, real HTTP integration, docs, API
   compatibility, lint, Staticcheck, and vulnerability checks.
3. Review `go list -m all`, `NOTICE`, third-party licenses, threat model,
   benchmark output, and changelog.
4. Update `api/baseline.txt` only for intentional reviewed API changes.
5. Create a signed annotated semantic-version tag.
6. Publication is maintainer-owned: the existing CI workflow provides
   verification and selected release rehearsal, not automatic tag publication.
   Publish the reviewed source archive, checksum and signed Sigstore provenance
   attestation only after the release gates pass. Never bypass failed
   verification; verify the actual public tag, assets and module consumer
   before describing a release as available.

Repository topics should include `go`, `golang`, `http`, `middleware`,
`net-http`, `security`, and `observability`.
