# http-middleware documentation map

[Repository README](../README.md) · [Current Golib portal](https://github.com/faustbrian/go-library-tools/blob/main/docs/ecosystem/README.md) · [Published ecosystem snapshot](https://github.com/faustbrian/go-library-tools/blob/v1.8.4/docs/ecosystem/README.md)

This map links package-owned adoption and operational guidance. Start with
the repository README for purpose, installation, and ownership boundaries.
Published compatibility sets are frozen multi-package baselines; they do not
certify every current main revision or a deployed application.

## Module boundaries

Import only the modules needed by your application. A deprecated facade,
preview adapter, or internal harness is not interchangeable with a stable
root API; follow its own README and migration guidance before composing it.

| Module | Status | Owner detail |
| --- | --- | --- |
| `github.com/faustbrian/go-http-middleware/v2` | stable | [README](../README.md) |
| `github.com/faustbrian/go-http-middleware/integration/siblings` | internal; non-releasable | [README](../integration/siblings/README.md) |

## Choose and adopt

- [Documentation](README.md).
- [Adoption guides](adoption.md).
- [API reference](api.md).
- [Architecture](architecture.md).
- [Behavior tables](behavior.md).
- [Compatibility](compatibility.md).
- [Cookbook](cookbook.md).
- [Dependency audit](dependencies.md).
- [FAQ](faq.md).
- [Integration cookbook](integrations.md).
- [Migration guide](migration.md).
- [Ordering reference](ordering.md).
- [Ownership and sibling integration](ownership.md).
- [ResponseWriter compatibility](responsewriter.md).
- [Standards scope and divergences](standards.md).

## Operate and evaluate boundaries

- [Security deployment](security.md).
- [Threat model and resource budgets](threat-model.md).
- [Troubleshooting](troubleshooting.md).

## Owned adapters, examples, and harnesses

- [Sibling interoperability harness](../integration/siblings/README.md).

## Maintainer, specification, and evidence references

- [Performance](performance.md).
- [Release process](release.md).
- [Source documentation audit](source-documentation-audit.md).
- [HTTP Middleware Specification Decisions](specification-decisions.md).
- [Verification evidence](verification.md).
- [Specification conformance matrix](../specification/README.md).

## Related next steps

[Contributor setup and checks](../CONTRIBUTING.md), [Compatibility policy](../COMPATIBILITY.md), [Release history](../CHANGELOG.md), [Private vulnerability reporting](../SECURITY.md), [Support](../SUPPORT.md).
