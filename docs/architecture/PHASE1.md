# Hearth — Phase 1 Specification
### Core Foundation

> **Goal**: A working CLI that can create accounts, record journal entries,
> enforce GAAP rules, and query balances — backed by SQLite, no server required.
> Everything in Phase 1 must be tested before or alongside implementation.

---

## Acceptance Criteria

Phase 1 is complete when all of the following are true:

- [ ] `go build ./...` succeeds with zero warnings
- [ ] `go test -race ./...` passes with 90%+ coverage on `internal/core/`
- [ ] `golangci-lint run` reports zero issues
- [ ] The CLI commands listed below execute correctly against a SQLite database
- [ ] All GAAP guard rules are covered by table-driven tests
- [ ] The Store interface is satisfied by the SQLite implementation
- [ ] The PostgreSQL implementation compiles (stubs acceptable) but is not required to pass tests

---

## Task List

Work through these in order. Each task has explicit acceptance criteria.
Do not start a task until the previous one's tests pass.

---

### Task 1.1 — Repository Scaffold

**What to create:**
- `go.mod` with module name `github.com/hearth-ledger/hearth` and Go 1.23
- All directories in the project structure (empty, with `.gitkeep`)
- `Makefile` with targets: `build`, `test`, `lint`, `clean`, `migrate-sqlite`
- `.golangci.yml` with sensible defaults (errcheck, govet, staticcheck, unused, gofmt)
- `.gitignore` appropriate for Go
- `README.md` with project name, one-paragraph description, and build instructions

**Acceptance criteria:**
- `make build` produces binaries for `hearth` and `hearthd`
- `make test` runs the test suite
- `make lint` runs golangci-lint

---

### Task 1.2 — Core Domain Types

**What to create:** `internal/core/` domain types with no external dependencies.

**Types to define:**

```go
// internal/core/currency/currency.go
type Currency string          // ISO 4217 code e.g. "USD"
type Amount struct {
    Value    decimal.Decimal
    Currency Currency
}

// internal/core/account/account.go
type AccountType  string      // asset | liability | equity | income | expense
type AccountID    string      // UUID
type Account struct { ... }

// internal/core/journal/journal.go
type EntryID     string       // UUID
type PostingID   string       // UUID
type JournalEntry struct { ... }
type Posting struct { ... }

// internal/core/envelope/envelope.go
type EnvelopeID  string
type Envelope struct { ... }
type Allocation struct { ... }
```

**Rules:**
- All IDs are UUID strings (use `google/uuid`)
- All monetary fields use `Amount` — never raw `decimal.Decimal` without a currency
- All structs are immutable value types where possible (no pointer receivers on value methods)
- No database tags on domain types — those belong in the store layer

**Acceptance criteria:**
- Types compile
- Unit tests for `Amount` arithmetic (add, subtract, compare across currencies — the last must return an error)

---

### Task 1.3 — GAAP Guard

**What to create:** `internal/core/gaap/`

**Rules to implement** (each as a named function returning a typed error):

| Rule | Function name | Error type |
|---|---|---|
| Entry must have ≥ 2 postings | `ValidateMinimumPostings` | `ErrInsufficientPostings` |
| Sum of all postings equals zero | `ValidateBalance` | `ErrUnbalancedEntry` |
| No posting has a zero amount | `ValidateNonZeroAmounts` | `ErrZeroAmountPosting` |
| All accounts belong to same household | `ValidateHouseholdConsistency` | `ErrCrossHouseholdEntry` |
| Entry is not posted to a locked period | `ValidatePeriodNotLocked` | `ErrLockedPeriod` |

**Guard function:**
```go
func Validate(entry JournalEntry, ctx ValidationContext) []ValidationError
```

Returns ALL violations, not just the first. A `ValidationError` wraps the rule name,
the error type, and a recovery hint string.

**Acceptance criteria:**
- Table-driven tests cover every rule
- Each rule has at least: valid case, single violation, edge cases
- Test names follow `TestValidate_<RuleName>_<Scenario>` convention
- `go test -race` passes

---

### Task 1.4 — Store Interface

**What to create:** `internal/store/store.go`

Define the complete Store interface for Phase 1 operations only:

```go
type Store interface {
    // Household
    CreateHousehold(ctx context.Context, h core.Household) error
    GetHousehold(ctx context.Context, id core.HouseholdID) (core.Household, error)

    // Accounts
    CreateAccount(ctx context.Context, a core.Account) error
    GetAccount(ctx context.Context, id core.AccountID) (core.Account, error)
    ListAccounts(ctx context.Context, householdID core.HouseholdID) ([]core.Account, error)

    // Journal
    CreateJournalEntry(ctx context.Context, e core.JournalEntry) error
    GetJournalEntry(ctx context.Context, id core.EntryID) (core.JournalEntry, error)
    ListJournalEntries(ctx context.Context, q JournalQuery) ([]core.JournalEntry, error)

    // Balances (computed)
    GetAccountBalance(ctx context.Context, id core.AccountID, asOf time.Time) (core.Amount, error)

    // Periods
    CreateFiscalPeriod(ctx context.Context, p core.FiscalPeriod) error
    LockFiscalPeriod(ctx context.Context, id core.PeriodID) error
}
```

**Acceptance criteria:**
- Interface is defined and compiles
- `JournalQuery` struct supports filtering by account, date range, description text
- No implementation yet — just the interface and its supporting types

---

### Task 1.5 — SQLite Migrations

**What to create:** `migrations/sqlite/` using goose

Migration files (in order):

```
0001_create_households.sql
0002_create_members.sql
0003_create_accounts.sql
0004_create_fiscal_periods.sql
0005_create_journal_entries.sql
0006_create_postings.sql
0007_create_envelopes.sql
0008_create_envelope_allocations.sql
0009_create_sync_events.sql
```

**Requirements:**
- All tables include `created_at` and `updated_at` (managed by triggers in SQLite)
- All monetary amounts stored as TEXT (decimal string) — never REAL or FLOAT
- All IDs stored as TEXT (UUID strings)
- Foreign key constraints enabled (`PRAGMA foreign_keys = ON` in connection setup)
- WAL mode enabled (`PRAGMA journal_mode = WAL`)
- Every migration has a corresponding `-- +goose Down` section

**Acceptance criteria:**
- `make migrate-sqlite` runs all migrations against a test database without error
- `make migrate-sqlite` is idempotent (safe to run twice)

---

### Task 1.6 — SQLite Store Implementation

**What to create:** `internal/store/sqlite/`

Implement the `Store` interface against SQLite using `sqlc`-generated queries where
possible, hand-written queries where `sqlc` falls short.

**Requirements:**
- Connection setup must enable WAL mode and foreign keys on every connection
- All queries use parameterized statements (no string interpolation)
- Decimal amounts marshalled as strings to/from the database
- `CreateJournalEntry` must be transactional — it inserts the entry and all postings
  atomically, or rolls back entirely

**Acceptance criteria:**
- All `Store` interface methods implemented
- Integration tests using `testcontainers-go` (or a temp file for SQLite) cover:
  - Happy path for every method
  - Foreign key violation returns a typed error, not a raw SQLite error
  - Creating a journal entry with a failing posting rolls back the whole entry

---

### Task 1.7 — CLI: Core Commands

**What to create:** `cmd/hearth/` using cobra

**Commands to implement:**

```
hearth init                    # Create a new household database
hearth accounts list           # List all accounts (table/json/csv)
hearth accounts add            # Interactive prompt to add an account
hearth accounts show <id>      # Show account detail + current balance
hearth transactions add        # Interactive double-entry transaction form
hearth transactions list       # List entries (filterable by account, date)
hearth transactions show <id>  # Show entry with all postings
hearth report balance          # Balance sheet as of today (or --as-of date)
hearth version                 # Print version info
hearth help                    # Standard cobra help
```

**Requirements:**
- All list/report commands support `--output [table|json|csv|plain]`
- `hearth init` creates `~/.local/share/hearth/ledger.db` by default;
  `--db` flag overrides the path
- `HEARTH_DB` environment variable sets the database path (overridden by `--db`)
- `hearth transactions add` runs the GAAP guard before writing;
  on failure, displays the full structured error with recovery hints
- Exit codes: 0 success, 1 user error, 2 system error, 3 GAAP violation

**Acceptance criteria:**
- `go test -race` passes on CLI layer tests (use `os/exec` to test full command execution)
- `hearth help` output is clean and documents every flag
- `hearth transactions add` with an unbalanced entry exits with code 3 and a helpful message

---

### Task 1.8 — Error Package

**What to create:** `pkg/errors/`

```go
type HearthError struct {
    Code     ErrorCode
    Message  string          // what happened
    Context  string          // why it happened
    Hints    []string        // numbered recovery options
    HelpTopic string         // hearth help <topic>
}

func (e *HearthError) Error() string
func (e *HearthError) UserFacing() string  // formatted for terminal output
```

**Error codes to define (at minimum):**
`ErrGAAPBalance`, `ErrGAAPMinPostings`, `ErrGAAPLockedPeriod`,
`ErrAccountNotFound`, `ErrHouseholdNotFound`, `ErrDatabaseConnection`,
`ErrInvalidAmount`, `ErrCurrencyMismatch`

**Acceptance criteria:**
- Unit tests for `UserFacing()` output format
- All GAAP guard errors use this package
- All store errors are wrapped into this package before surfacing to CLI

---

### Task 1.9 — ADRs for Phase 1 Decisions

**What to create:** `docs/architecture/adr/`

Write the following ADRs before Phase 1 work begins (or at least before the
relevant task):

- `ADR-001-go-as-primary-language.md`
- `ADR-002-event-sourcing-for-ledger.md`
- `ADR-003-sqlite-and-postgres-dual-store.md`
- `ADR-004-decimal-string-storage.md`  ← why amounts are TEXT in SQLite

---

## What Phase 1 Explicitly Does Not Include

Do not implement, stub, or reference these in Phase 1 code:

- Authentication or JWT
- The REST API or `hearthd` server (binary scaffolded, but empty)
- Envelope commands
- Import adapters
- AI harness
- Sync engine
- TUI
- Multi-member household (single household owner only in Phase 1)

---

## Definition of Done

A PR (or completed session) for Phase 1 is done when:

1. `make build` — clean
2. `make test` — all pass, race detector clean, ≥90% coverage on `internal/core/`
3. `make lint` — zero issues
4. `make migrate-sqlite` — idempotent, clean
5. All 9 task acceptance criteria above are met
6. Four ADRs are written
7. `README.md` has accurate build and quickstart instructions
