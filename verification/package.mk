.PHONY: conformance interoperability

conformance:
	./scripts/check-conformance.sh

interoperability:
	cd integration/siblings && GOWORK=off go test ./... -count=1
