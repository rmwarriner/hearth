# ADR-006: OpenAPI Spec-First with oapi-codegen

## Status
Accepted

## Context
The REST API needs a machine-readable contract for three consumers: the Go server handlers, the future iOS client, and developer documentation. Two approaches were considered:

1. **Code-first** — annotate Go types and handlers with struct tags or reflection; generate the OpenAPI spec from Go source at build time (e.g., `ogen`, `swaggo`).
2. **Spec-first** — hand-write `docs/openapi.yaml` as the source of truth; use `oapi-codegen` to generate Go server stubs and request/response types from the spec.

The `hearth-architecture.md` document already states: *"OpenAPI spec is the source of truth."* Spec-first is the stated intent.

Code-first was rejected for two reasons:
- Annotation-heavy code entangles business logic with HTTP contract concerns, making handlers harder to test in isolation.
- The generated spec is a side-effect of the Go types, which means the iOS client's contract is implicitly defined by Go implementation details rather than an explicit design artifact.

## Decision

`docs/openapi.yaml` (OpenAPI **3.0.3**) is the single source of truth for the API contract. It is hand-written.

Note: OpenAPI 3.1 was evaluated but `oapi-codegen` v2 does not yet fully support 3.1 schema semantics (nullable types, `const`, JSON Schema 2020-12 vocabulary). The spec uses 3.0.3, which is fully supported. This can be revisited when oapi-codegen adds stable 3.1 support.

`oapi-codegen` is used as a **dev-time codegen tool** (equivalent to `sqlc`) to generate:
- Request/response types (`internal/api/openapi/api.gen.go`)
- A chi-compatible server interface that handlers must implement

`oapi-codegen` is recorded in `tools.go` with a `//go:build tools` constraint so `go mod tidy` tracks it without making it a runtime import. The generated file is committed to the repository (same policy as sqlc-generated files).

Note: `oapi-codegen` is not on the originally approved dependency list in `CLAUDE.md`. It is approved here as a dev-time codegen tool with no runtime import, equivalent in kind to `sqlc`.

## Consequences

- The spec document in `docs/openapi.yaml` is the contract for both the server and the iOS client. API changes require updating the spec first, then regenerating, then updating handlers.
- `make generate-api` must be re-run whenever `docs/openapi.yaml` changes. CI should verify the generated file is up to date (compare `git diff` after regeneration).
- Handlers remain hand-written and thin. The generated interface enforces that all spec endpoints are implemented at compile time — missing a handler is a build error.
