# Sibling interoperability harness

This internal, non-releasable module verifies that `http-middleware` composes
with the independently released Golib HTTP service stack. It is an engineering
verification harness, not a public library or an installation target.

The tests prove three boundaries:

- `siblings_test.go` checks bounded router observation metadata and rejects
  duplicate ownership of recovery, request identifiers, and body limits when
  `service/serverhttp` already owns those concerns. It also exercises Correlation
  HTTP request identity, explicit peer trust, and malformed-header rejection.
- `platform_test.go` exercises an in-memory service composition spanning the
  HTTP client, logging, authentication, authorization, JSON:API, JSON-RPC,
  OpenAPI, routing, and service HTTP composition.
- `authentication_v2_test.go` composes the actual v2 bearer middleware and
  structural authorization mapper, preserving authenticated identity and
  preventing missing or rejected credentials from reaching the application.

All sibling module versions are pinned in [`go.mod`](go.mod). The harness uses
in-memory handlers and `httptest`; it does not require a deployed service or
other external runtime.

The middleware, HTTP client, and Authentication dependencies use their `/v2` modules.
Authentication imports move together to preserve nominal type identity; its
HTTP integration uses the canonical `adapters/http` package.
Correlation uses its canonical `adapters/http` package and identifier v2
generation. Other independently released siblings retain their pinned versions.

Run the contract for every declared module from the repository root:

```sh
make check
```

For the ownership model and composition guidance, see the
[sibling integration contract](../../docs/ownership.md) and
[integration cookbook](../../docs/integrations.md).
