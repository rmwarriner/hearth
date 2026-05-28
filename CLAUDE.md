# Hearth — Claude Code Briefing

This file is the authoritative context document for AI-assisted development of Hearth.
Read it fully before writing any code. When in doubt, ask rather than assume.

---

## What This Project Is

Hearth is a cloud-native household accounting system. It is a clean-slate successor to
GnuCash. It enforces double-entry bookkeeping (GAAP-compliant) with an envelope budgeting
view layer on top. It is designed for a household with multiple members.

Full architecture details: `docs/architecture/hearth-architecture.md`

---

## Non-Negotiables

These constraints must never be violated. If a task seems to require violating one,
stop and say so rather than working around it silently.

1. **No floating-point arithmetic for money.** All monetary values use `shopspring/decimal`.
   Never use `float32`, `float64`, or `int` to represent currency amounts.

2. **Journal entries are immutable.** No UPDATE or DELETE on `journal_entries` or `postings`.
   Corrections are made via reversing entries only.

3. **The GAAP guard runs on every journal entry before persistence.** No bypass path exists.
   If you add a fast path that skips validation, it will be rejected.

4. **Tests are written before or alongside implementation** (TDD). Do not write an
   implementation file without a corresponding test file.

5. **No PII or financial amounts in log output.** Use structured logging (zerolog).
   Log the operation and its outcome; never log payee names, amounts, or account names
   at any log level.

6. **Errors must include recovery hints.** See the error design section below.

---

## Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.23+ |
| CLI framework | cobra + viper |
| TUI framework | bubbletea + lipgloss + huh |
| HTTP router | chi v5 |
| Auth | golang-jwt/jwt v5 |
| Decimal math | shopspring/decimal |
| Local database | SQLite via modernc.org/sqlite (pure Go, no CGO) |
| Server database | PostgreSQL 16+ |
| Migrations | pressly/goose v3 |
| SQL codegen | sqlc (dev tool, not a runtime dep) |
| Logging | rs/zerolog |
| Tracing | go.opentelemetry.io/otel |
| Testing | testify + testcontainers-go + teatest |

Do not introduce dependencies outside this list without flagging it first.
If a better option exists, propose it with rationale — don't silently add it.

---

## Project Structure

```
hearth/
├── CLAUDE.md                   ← this file
├── cmd/
│   ├── hearth/main.go          ← CLI entrypoint
│   └── hearthd/main.go         ← Server daemon entrypoint
├── internal/
│   ├── core/                   ← Pure domain logic — no I/O, no database
│   │   ├── account/
│   │   ├── journal/
│   │   ├── envelope/
│   │   ├── gaap/               ← Validation rules, GAAP guard
│   │   └── currency/
│   ├── store/                  ← Storage abstraction layer
│   │   ├── store.go            ← Store interface (the contract)
│   │   ├── sqlite/
│   │   └── postgres/
│   ├── api/                    ← REST API (server mode only)
│   │   ├── handler/
│   │   ├── middleware/
│   │   └── openapi/
│   ├── ai/                     ← AI harness (tiered)
│   │   ├── rules/              ← Tier 0: no external calls
│   │   ├── inference/          ← Tier 1: private endpoint
│   │   ├── external/           ← Tier 2: external provider
│   │   └── privacy/            ← Scrubber, policy, field permissions
│   ├── importer/
│   │   ├── ofx/
│   │   ├── csv/
│   │   ├── qif/
│   │   ├── simplefin/          ← Primary bank feed connector
│   │   ├── plaid/              ← Stub only — future integration
│   │   └── gnucash/
│   ├── sync/
│   └── tui/
├── pkg/
│   ├── errors/                 ← Typed errors with recovery hints
│   └── event/
├── migrations/
│   ├── sqlite/
│   └── postgres/
├── deploy/
│   ├── docker-compose.yml
│   ├── Dockerfile
│   └── k8s/
├── docs/
│   ├── architecture/
│   │   ├── ARCHITECTURE.md
│   │   └── adr/                ← Architecture Decision Records
│   └── man/
└── tests/
    ├── unit/
    ├── integration/
    └── e2e/
```

`internal/` is private to this module. `pkg/` is the stable public extension surface.

---

## The Store Interface

This is the most important interface in the project. Everything depends on it.
It must be defined before any implementation is written.

The Store interface in `internal/store/store.go` must:
- Accept a `context.Context` as the first argument on every method
- Return typed errors (not raw `error` strings)
- Have both a SQLite implementation and a PostgreSQL implementation
- Be fully tested against both backends via `testcontainers-go`

When adding a new capability that touches the database, define the interface method first,
then write the test, then write the two implementations.

---

## Error Design

Every error returned to the user (CLI output, API response, TUI message) must follow
this structure. This is enforced by the types in `pkg/errors/`.

```
Error: <what happened — one sentence, plain English>
  <why it happened — one sentence of context>

  To fix this, you can:
    1. <concrete action>
    2. <concrete action>

  Learn more: hearth help <topic>
```

Internal errors (between packages) use standard Go error wrapping with `fmt.Errorf` and `%w`.
Only errors that surface to a user go through `pkg/errors/`.

---

## GAAP Guard

Location: `internal/core/gaap/`

The guard validates every `JournalEntry` before it is passed to the store.
Rules to enforce:

1. Sum of all postings must equal zero (debits + credits balance)
2. Entry must have at least two postings
3. All accounts referenced must exist and belong to the same household
4. Entry cannot be posted to a locked fiscal period
5. All postings in a multi-currency entry must include an exchange-rate posting

Each rule is a separate function with a name that makes the test output self-documenting.
Rules are composed — the guard runs all of them and collects all violations before returning,
so the user sees every problem at once, not one at a time.

---

## AI Tier Indicator

Any command or TUI screen that invokes AI must display the active tier:

- `[AI: OFF]` — Tier 0, rules engine
- `[AI: PRIVATE ●]` — Tier 1, private endpoint
- `[AI: CLOUD ⚠ ]` — Tier 2, external AI active

The Tier 2 indicator is shown in amber/yellow. It is never suppressed.
On CLI, it appears as a prefix line before command output when Tier 2 is active.
On TUI, it appears in the persistent status bar.

---

## Testing Conventions

- Unit tests live alongside the code they test (`account_test.go` next to `account.go`)
- Integration tests that require a database live in `tests/integration/`
- Use `testcontainers-go` to spin up real SQLite and PostgreSQL instances in integration tests
- Use table-driven tests for all validation logic (GAAP rules, import parsing, etc.)
- Run `go test -race ./...` — all tests must pass with the race detector enabled
- Target: 90%+ coverage on `internal/core/`

Test function naming: `TestFunctionName_Scenario_ExpectedOutcome`
Example: `TestGAAPGuard_UnbalancedEntry_ReturnsBalanceError`

---

## Logging Conventions

Use `rs/zerolog`. Log levels:

| Level | When to use |
|---|---|
| `Error` | Operation failed, user action required |
| `Warn` | Operation succeeded but something is notable |
| `Info` | Normal significant operations (startup, sync complete) |
| `Debug` | Internal state useful for debugging (disabled in production) |

Never log: payee names, account names, amounts, balances, user email addresses,
member names, or any data from `posting`, `journal_entry`, or `account` rows.
Log operation types and IDs only.

---

## CLI Conventions

- Every command that produces output supports `--output [table|json|csv|plain]`
- `table` is the default; it is human-readable and toner-friendly when printed
- `json` and `csv` outputs are machine-readable and pipeable
- Every command supports `--household` to override the active household (for scripting)
- Long-running commands show progress (use `bubbletea` spinner or `lipgloss` progress bar)
- Exit codes: 0 = success, 1 = user error, 2 = system error, 3 = GAAP violation

---

## Currency Handling

Every monetary amount stores both a decimal value and a currency code.
The household has a `base_currency` (default: USD).

Use `shopspring/decimal.Decimal` for all arithmetic.
Never round until display time. When rounding is required for display, use banker's rounding.

---

## What Is Not Built Yet (Do Not Stub or Fake)

The following are planned for later phases. Do not write placeholder implementations
that silently do nothing — leave them as explicit `// TODO(phase-N):` comments or
return a `ErrNotImplemented` error with a clear message.

- SimpleFIN Bridge connector (Phase 4)
- Plaid / Yodlee connectors (future, post Phase 4)
- AI tiers 1 and 2 (Phase 5)
- iOS client (Phase 6)
- PDF report rendering (Phase 7)
- Local↔Server sync (Phase 4)

---

## Architecture Decision Records

Before implementing anything non-obvious, check `docs/architecture/adr/` for a relevant ADR.
If none exists and the decision is significant, write the ADR first.

ADR format: `docs/architecture/adr/ADR-NNN-short-title.md`

```markdown
# ADR-NNN: Short Title

## Status
Accepted

## Context
Why does this decision need to be made?

## Decision
What was decided?

## Consequences
What are the trade-offs?
```

---

## Current Phase

**Phase 1 — Core Foundation**

See `docs/architecture/PHASE1.md` for the specific task list and acceptance criteria.
Do not work outside Phase 1 scope without explicit instruction.
