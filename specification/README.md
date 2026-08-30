# Specification conformance matrix

The [specification decision register](../docs/specification-decisions.md),
`decisions.json`, `conformance.json`, `decision-history.json`, and
`monitoring.json` define the auditable contract backed by `manifest.tsv`.
The module claims only its documented public policy surface and delegates HTTP
transport to Go.

| Decision | Policy |
|---|---|
| `HTTPMIDDLEWARE-DEC-001` | Explicit composition order and duplicate ownership |
| `HTTPMIDDLEWARE-DEC-002` | Request and correlation identifier trust |
| `HTTPMIDDLEWARE-DEC-003` | Panic recovery after response commitment |
| `HTTPMIDDLEWARE-DEC-004` | Request body limit accounting and ownership |
| `HTTPMIDDLEWARE-DEC-005` | Deadlines versus buffered handler timeouts |
| `HTTPMIDDLEWARE-DEC-006` | Forwarded fields and trusted-hop selection |
| `HTTPMIDDLEWARE-DEC-007` | CORS origins, wildcards, and preflight ownership |
| `HTTPMIDDLEWARE-DEC-008` | Security header policy and HSTS acknowledgement |
| `HTTPMIDDLEWARE-DEC-009` | Content-coding negotiation and transformed metadata |
| `HTTPMIDDLEWARE-DEC-010` | Request and response media negotiation |
| `HTTPMIDDLEWARE-DEC-011` | ResponseWriter capabilities and commitment |
| `HTTPMIDDLEWARE-DEC-012` | Completion observation and privacy |
| `HTTPMIDDLEWARE-DEC-013` | Local admission, fairness, and retry guidance |
| `HTTPMIDDLEWARE-DEC-014` | Cache and maintenance response policy |
| `HTTPMIDDLEWARE-DEC-015` | Concern ownership and sibling integration |

Run `golib specification check` offline and add `--online` to verify reviewed
authority content and change-monitor pins.
