# Phase 2 — Server Mode

## Overview

Phase 2 delivers the `hearthd` REST API daemon on top of the Phase 1 core foundation.
It adds PostgreSQL persistence, JWT authentication with rotating refresh tokens,
an OpenAPI-spec-first REST API, structured logging with PII redaction, and Docker deployment.

**Status: Complete** — 2026-05-28

---

## Acceptance Criteria

- [x] `go build ./...` succeeds with zero warnings
- [x] `go test -race ./...` passes on all non-integration packages
- [x] `golangci-lint run` reports zero issues on all new packages
- [x] `hearthd` binary starts, listens on `:8080`, responds to `/api/v1/auth/login`
- [x] JWT login → refresh → logout flow works end-to-end
- [x] Balanced journal entry returns 201; unbalanced returns 422 with violations list
- [x] `hearthd` binary has no dependency on `internal/store/sqlite`
- [x] `docker compose up --build` starts cleanly
- [x] All IDOR vulnerabilities addressed (household ownership verified on all by-ID lookups)
- [x] PostgreSQL RLS enabled with FORCE + WITH CHECK on all household-scoped tables

---

## Tasks Completed

| Task | Description |
|---|---|
| 2.1 | ADRs (JWT, OpenAPI, RLS strategy) + 9 new error codes |
| 2.2 | `internal/core/member` domain type |
| 2.3 | Store interface extended (members, refresh tokens); SQLite + PostgreSQL stubs |
| 2.4 | PostgreSQL migrations (11 files, including RLS with FORCE + WITH CHECK) |
| 2.5 | PostgreSQL connection setup (pgxpool, goose auto-migrations, SetHouseholdContext) |
| 2.6 | PostgreSQL store implementation (all Store methods, pgconn error mapping) |
| 2.7 | Logging infrastructure (zerolog + PII redaction for all JSON value types) |
| 2.8 | OpenAPI 3.0.3 spec + oapi-codegen generation (27 endpoints) |
| 2.9 | Auth service (JWT HS256, rotating refresh tokens, family revocation on replay) |
| 2.10 | API middleware (Authenticate, VerifyHousehold, RequestLogger, Recovery) |
| 2.11 | API handlers (thin handlers, GAAP guard on CreateJournalEntry) |
| 2.12 | `hearthd` daemon (config, wire, graceful shutdown, SIGTERM/SIGINT) |
| 2.13 | Docker deployment (multi-stage Dockerfile, docker-compose.yml) |
| 2.14 | Makefile targets + golangci.yml updates |
| 2.15 | This document + architecture doc v0.3 |

---

## Key Architecture Decisions

### JWT Auth Model (ADR-005)
Short-lived access tokens (15 min, HS256, stateless) paired with DB-backed rotating
refresh tokens. SHA-256 stored (not bcrypt) — the raw token is 32 bytes of crypto-random,
making SHA-256 safe as a deterministic lookup key. Replay detection via token family
revocation: presenting a used token revokes the entire family.

### OpenAPI Spec-First (ADR-006)
`docs/openapi.yaml` is the source of truth. `oapi-codegen` generates `ServerInterface`
as a dev tool (not a runtime dependency). The generated file is committed. The iOS client
(Phase 6) consumes the same spec.

### PostgreSQL RLS Strategy (ADR-007)
Every write transaction calls `SET LOCAL app.household_id = $1` before the first query.
All household-scoped tables have `FORCE ROW LEVEL SECURITY` (so the table owner is also
subject to policies) and `FOR ALL ... WITH CHECK (...)` policies (so writes are also
enforced at the DB layer). Defense-in-depth alongside the Go-layer household checks.

### IDOR Hardening
Beyond the `VerifyHousehold` middleware (which verifies JWT claim matches URL parameter),
every handler that loads a resource by raw ID verifies the loaded resource's `HouseholdID`
matches the URL parameter before responding. Cross-household writes are additionally
blocked by RLS WITH CHECK policies.

---

## Route Structure

```
POST   /api/v1/auth/login          (unauthenticated)
POST   /api/v1/auth/refresh        (unauthenticated)
POST   /api/v1/auth/logout         (authenticated)

GET    /api/v1/households/{id}
PATCH  /api/v1/households/{id}

GET    /api/v1/households/{id}/members
POST   /api/v1/households/{id}/members          (owner only)
GET    /api/v1/households/{id}/members/{mid}
PATCH  /api/v1/households/{id}/members/{mid}    (owner only)
DELETE /api/v1/households/{id}/members/{mid}    (501 — Phase 3)

GET    /api/v1/households/{id}/accounts
POST   /api/v1/households/{id}/accounts
GET    /api/v1/households/{id}/accounts/{aid}
PATCH  /api/v1/households/{id}/accounts/{aid}
GET    /api/v1/households/{id}/accounts/{aid}/balance

GET    /api/v1/households/{id}/entries
POST   /api/v1/households/{id}/entries
GET    /api/v1/households/{id}/entries/{eid}
POST   /api/v1/households/{id}/entries/{eid}/reverse

GET    /api/v1/households/{id}/periods
POST   /api/v1/households/{id}/periods
POST   /api/v1/households/{id}/periods/{pid}/lock

GET    /api/v1/households/{id}/reports/balance-sheet
GET    /api/v1/households/{id}/reports/income-statement
```

---

## Definition of Done

**Phase 2 is complete as of 2026-05-28.** All criteria met:

1. `make build` — clean ✓
2. `make lint` — zero issues on all Phase 2 packages ✓
3. `go build ./...` — no sqlite dependency in hearthd ✓
4. PostgreSQL RLS with FORCE + WITH CHECK on all household-scoped tables ✓
5. JWT token family revocation on replay ✓
6. PII redaction in logs handles all JSON value types ✓
7. IDOR hardening: every by-ID lookup verifies household ownership ✓
8. Three ADRs written (ADR-005, ADR-006, ADR-007) ✓
9. `deploy/Dockerfile` (multi-stage, CGO_ENABLED=0) + `docker-compose.yml` ✓
