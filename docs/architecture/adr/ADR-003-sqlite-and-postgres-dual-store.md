# ADR-003: SQLite and PostgreSQL Dual Store

## Status
Accepted

## Context
Hearth has two target deployment modes: offline-first local mode (single user, no network) and cloud-native server mode (multi-user, multi-device sync). These modes have different database requirements. A single database target would force compromises: requiring PostgreSQL for local mode adds operational burden; using SQLite for server mode sacrifices multi-user isolation and row-level security.

## Decision
The storage layer is abstracted behind a `Store` interface defined in `internal/store/store.go`. Two implementations are maintained:
- `internal/store/sqlite/` — for local mode, using `modernc.org/sqlite` (CGO-free pure Go)
- `internal/store/postgres/` — for server mode, using `pgx/v5`

Migrations are kept separate per backend (`migrations/sqlite/` and `migrations/postgres/`). The `Store` interface is the contract; both implementations must pass the same integration test suite via `testcontainers-go`.

## Consequences
- SQLite uses WAL mode and has foreign keys enabled on every connection open.
- All monetary amounts are stored as TEXT (decimal strings) in both backends, preventing floating-point precision loss at the storage layer. See ADR-004.
- The SQLite implementation is the Phase 1 requirement; the PostgreSQL implementation compiles in Phase 1 but full test coverage is Phase 2.
- Adding a new database capability requires: define the interface method → write the test → implement in SQLite → implement in PostgreSQL. This order is enforced by convention.
- `sqlc` generates type-safe query functions from `.sql` files; the generated code is committed so `sqlc` is a dev-time tool, not a runtime dependency.
