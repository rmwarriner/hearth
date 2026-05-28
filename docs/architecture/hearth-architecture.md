# Hearth — Architecture & Design Document
### A Cloud-Native Household Accounting System

> **Status**: Phase 1 complete — CLI + SQLite core operational  
> **Targets**: macOS · Linux · iOS  
> **Paradigm**: Cloud-native, offline-capable, privacy-first  
> **Build Method**: AI tooling (Claude Code)

---

## 1. Project Name

**Hearth** — the center of a household. Accounting as infrastructure, not software.  
CLI binary: `hearth` · Server daemon: `hearthd` · Config: `~/.config/hearth/`

---

## 2. Tech Stack — Rationale

### Primary Language: Go 1.23+

Go is the right choice for this project given the constraints:

- The **Charm.sh ecosystem** (`bubbletea`, `lipgloss`, `huh`, `glamour`) is the gold standard for CLI/TUI applications — purpose-built for exactly this use case and actively maintained
- `cobra` + `viper` is the CLI framework used by kubectl, Docker, Helm, and most serious Go CLIs — it produces hledger-style scriptable interfaces naturally
- **Single static binary** with no runtime dependencies — the offline-first CLI/TUI requirement is satisfied without a packaging problem
- Go generates exceptionally clean, readable, reviewable code from AI tooling
- **sqlc** (generates type-safe Go from raw SQL) paired with `goose` (migrations) creates a data layer that AI tools handle very well
- Cross-compiles cleanly to `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`
- `shopspring/decimal` handles financial fixed-point arithmetic correctly — no floating-point errors

### iOS: Swift / SwiftUI (separate codebase)

iOS requires Swift regardless of backend language. The iOS client is a thin consumer of the REST API — no shared code with the Go backend needed.

### Databases: SQLite (local) + PostgreSQL 16+ (server)

The offline-first + cloud-sync requirement demands two database targets:

| Mode | Database | Use case |
|---|---|---|
| Local | SQLite (via `modernc.org/sqlite` — pure Go, no CGO) | Single-user, no server |
| Server | PostgreSQL 16+ | Multi-user, cloud-native |

The storage layer is abstracted behind an interface. Migrations are kept separate per backend (`migrations/sqlite/`, `migrations/postgres/`). SQLite uses WAL mode for concurrent read performance.

### Container Runtime: Docker / Podman (OCI-compatible)

`docker-compose.yml` for local development and self-hosted deployment. Kubernetes manifests in `deploy/k8s/` for cloud deployment. No vendor lock-in.

### API: REST + OpenAPI 3.1

GraphQL was considered and rejected — the query complexity it introduces is not justified for a household-scale application, and AI tooling generates REST handlers more reliably. OpenAPI 3.1 spec is the source of truth, generated from Go types via `ogen` or `oapi-codegen`.

### AI Harness: Tiered Privacy Model

AI capabilities operate across three tiers of increasing capability and decreasing privacy. The default tier on first run is **Tier 0**. Advancing to a higher tier requires explicit user action.

| Tier | Name | What runs | Data leaves device? |
|---|---|---|---|
| 0 | Rules Engine | Coded heuristics + user rules + statistical methods | Never |
| 1 | Private Inference | Any OpenAI-compatible endpoint on a trusted network | No (LAN/VPN only) |
| 2 | External AI | Any external provider (OpenAI, Anthropic, etc.) | Yes — with explicit consent |

**Tier 0** is not a degraded fallback — it is a capable default. User-defined pattern/keyword/regex rules, statistical anomaly detection (z-score, IQR), and payee normalization handle the majority of household categorization needs without any model.

**Tier 1** accepts any OpenAI-compatible local endpoint: Ollama, LM Studio, Jan.ai, or a self-hosted inference server. Hearth does not require or assume any specific implementation.

**Tier 2** requires completing a **data permission setup** before activating — there is no path to enable external AI that bypasses the permission dialog.

#### Data Field Permissions (Tier 1 and 2)

Each field can be independently permitted or denied. Permissions apply globally and are further constrained per-feature:

| Field | Default (Tier 1) | Default (Tier 2) | Sensitivity |
|---|---|---|---|
| Transaction date | ✓ permitted | Prompt user | Low |
| Payee / merchant name | ✓ permitted | Prompt user | Medium |
| Transaction amount | ✓ permitted | **Denied by default** | High |
| Account names | ✓ permitted | **Denied by default** | High |
| Memo / notes text | ✓ permitted | Prompt user | Variable |
| Account balances | ✓ permitted | **Denied by default** | Very high |

If a feature's minimum field requirements exceed what the user has permitted, that feature silently falls back to Tier 0 for that call — it never errors, never prompts repeatedly, and never sends more than permitted.

#### Per-Feature Tier Override

Each AI feature can be independently assigned a tier, overriding the global setting:

```yaml
ai:
  tier: 2                              # global default
  private_endpoint: "http://192.168.1.x:8080/v1"
  external_provider: "openai"
  external_api_key_env: "HEARTH_OPENAI_KEY"

  data_permissions:                    # applies to Tier 2
    send_payee: true
    send_amounts: false                # never send amounts externally
    send_account_names: false
    send_memos: true
    send_dates: true
    send_balances: false

  features:
    categorization:
      enabled: true
      tier_override: 2                 # use external AI for payee matching
    anomaly_detection:
      enabled: true
      tier_override: 1                 # keep anomaly detection on private network
                                       # (needs amounts — not permitted externally)
    nl_queries:
      enabled: false                   # disabled until explicitly turned on
```

#### Visual Indicator

The active AI tier is displayed persistently — in the TUI status bar, prefixed on every CLI command that invokes AI, and as a startup banner when Tier 2 is active:

```
[AI: OFF]          # Tier 0 — rules engine, no color
[AI: PRIVATE ●]    # Tier 1 — green, private endpoint active  
[AI: CLOUD ⚠ ]    # Tier 2 — amber, always shown when external AI active
```

The Tier 2 startup banner is not suppressible. It shows the active data permissions summary on every launch so there is no ambiguity about what is configured.

---

## 3. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          CLIENTS                                 │
│                                                                  │
│   hearth CLI (cobra)    hearth TUI (bubbletea)    iOS (SwiftUI) │
│        │                        │                       │        │
│        │◄── Local mode ────────►│                       │        │
│        │    (SQLite, no server) │                       │        │
└────────┼────────────────────────┼───────────────────────┼────────┘
         │                        │                       │
         │   Direct store access  │             REST/HTTPS│
         │   (local mode)         │             (JWT auth)│
         │                        │                       │
┌────────▼────────────────────────▼───────────────────────▼────────┐
│                        CORE ENGINE (Go)                           │
│                                                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐  │
│  │  Accounting Core │  │  Envelope Engine  │  │  GAAP Guard    │  │
│  │                  │  │                  │  │                │  │
│  │ - Double-entry   │  │ - Budget periods  │  │ - Entry valid. │  │
│  │ - Account chart  │  │ - Allocation      │  │ - Balance check│  │
│  │ - Journal entries│  │ - Overspend alerts│  │ - Reversal only│  │
│  │ - Fiscal periods │  │ - Rollover rules  │  │ - Period lock  │  │
│  └──────────────────┘  └──────────────────┘  └────────────────┘  │
│                                                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐  │
│  │  Import Adapters │  │   AI Harness      │  │  Sync Engine   │  │
│  │                  │  │                  │  │                │  │
│  │ - OFX/QFX        │  │ T0: Rules engine │  │ - Local→Server │  │
│  │ - CSV (flexible) │  │ T1: Private infer│  │ - Conflict res.│  │
│  │ - QIF            │  │ T2: External AI  │  │ - Event replay │  │
│  │ - SimpleFIN      │  │ Field-level perms│  │                │  │
│  │ - GnuCash XML    │  │ Per-feature tier │  │                │  │
│  └──────────────────┘  └──────────────────┘  └────────────────┘  │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │                  Storage Abstraction                      │    │
│  │                                                           │    │
│  │   Store interface  ◄── SQLite impl  ◄── PostgreSQL impl  │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │                   REST API Layer                          │    │
│  │   chi router · JWT middleware · OpenAPI 3.1 spec          │    │
│  └──────────────────────────────────────────────────────────┘    │
└───────────────────────────────────────────────────────────────────┘
              │                          │
         SQLite file               PostgreSQL 16+
         (~/.local/share/         (Docker / cloud)
          hearth/ledger.db)
```

---

## 4. Data Model & Event Sourcing

### Why Event Sourcing

A double-entry ledger is *already* an event log by nature — transactions are facts, not state mutations. Event sourcing formalizes this:

- Every financial event is **appended**, never mutated or deleted
- Account balances, envelope allocations, and reports are **projections** computed from the event stream
- The full audit trail is implicit in the design, not bolted on
- GAAP immutability requirement is satisfied structurally, not by policy

Corrections are made via **reversing entries** — a new transaction that negates the original — exactly as GAAP requires.

### Core Entities

```
household
  id, name, fiscal_year_start, base_currency, created_at

member
  id, household_id, display_name, email, role (owner|member|viewer)
  password_hash, created_at

account
  id, household_id, name, type (asset|liability|equity|income|expense)
  subtype, currency, parent_id (nullable), is_placeholder, created_at

journal_entry  ← append-only, never updated
  id, household_id, posted_at, description, reference, source
  created_by (member_id), created_at, is_reversal_of (nullable)

posting        ← the two (or more) sides of each entry
  id, journal_entry_id, account_id, amount (decimal), currency
  memo

envelope       ← budget container, not an account
  id, household_id, name, target_amount, period_type, rollover_policy

envelope_allocation  ← append-only
  id, envelope_id, period_start, amount, created_at

envelope_transaction ← links postings to envelopes
  id, posting_id, envelope_id, amount

sync_event     ← local mode conflict resolution
  id, device_id, sequence, payload_type, payload, synced_at
```

### Currency Design

Every monetary amount (`posting.amount`) stores a decimal value and a currency code. The base currency is set at the household level. Conversion is handled at reporting time. This adds negligible complexity now and avoids a catastrophic migration later.

---

## 5. Household GAAP Enforcement

Hearth enforces a pragmatic subset of GAAP principles appropriate for household use. The GAAP Guard validates every journal entry before persistence:

| Principle | Enforcement |
|---|---|
| **Double-entry balance** | Sum of all postings in an entry must equal zero (debits + credits = 0) |
| **Account type validity** | Postings must reference accounts of the correct type for the transaction class |
| **Immutability** | No UPDATE or DELETE on `journal_entry` or `posting` — corrections require reversing entries |
| **Period awareness** | Entries cannot be posted to a locked fiscal period |
| **Account classification** | Asset, Liability, Equity, Income, Expense — enforced at account creation |
| **Currency consistency** | Multi-currency entries must have explicit exchange rate postings |
| **Non-negative assets** | Configurable warning (not hard block) when asset accounts go negative |

---

## 6. Project Structure

```
hearth/
├── cmd/
│   ├── hearth/             # CLI entrypoint (cobra root command)
│   │   └── main.go
│   └── hearthd/            # Server daemon entrypoint
│       └── main.go
│
├── internal/
│   ├── core/               # Pure accounting domain — no I/O
│   │   ├── account/        # Account types, chart of accounts
│   │   ├── journal/        # Journal entries, postings
│   │   ├── envelope/       # Budget envelopes, allocations
│   │   ├── gaap/           # Validation rules, GAAP guard
│   │   └── currency/       # Decimal arithmetic, conversion
│   │
│   ├── store/              # Storage abstraction
│   │   ├── store.go        # Store interface definition
│   │   ├── sqlite/         # SQLite implementation
│   │   └── postgres/       # PostgreSQL implementation
│   │
│   ├── api/                # REST API (chi router)
│   │   ├── handler/        # HTTP handlers (thin, delegate to core)
│   │   ├── middleware/     # Auth, logging, rate-limit
│   │   └── openapi/        # Generated OpenAPI spec
│   │
│   ├── ai/                 # AI harness
│   │   ├── harness.go      # Tier dispatch + feature routing
│   │   ├── rules/          # Tier 0: heuristics, user rules, statistical
│   │   ├── inference/      # Tier 1: private OpenAI-compatible endpoint
│   │   ├── external/       # Tier 2: external provider clients
│   │   ├── privacy/        # Field-level scrubber, permission policy
│   │   │   ├── policy.go   # Permission struct, tier definitions
│   │   │   └── scrubber.go # Redacts fields before outbound calls
│   │   ├── categorize.go   # Categorization feature (tier-aware)
│   │   └── anomaly.go      # Anomaly detection feature (tier-aware)
│   │
│   ├── importer/           # Import adapters
│   │   ├── ofx/
│   │   ├── csv/
│   │   ├── qif/
│   │   ├── simplefin/      # Primary: SimpleFIN Bridge
│   │   └── plaid/          # Future: Plaid / Yodlee (stubbed)
│   │   └── gnucash/
│   │
│   ├── sync/               # Local↔Server sync
│   │   ├── engine.go       # Sync orchestration
│   │   └── conflict.go     # Conflict resolution
│   │
│   └── tui/                # Bubbletea TUI
│       ├── app.go          # Root model
│       ├── dashboard/
│       ├── accounts/
│       ├── transactions/
│       └── envelopes/
│
├── pkg/                    # Importable by extensions
│   ├── errors/             # Typed errors with recovery hints
│   └── event/              # Event types for sync
│
├── migrations/
│   ├── sqlite/             # goose migrations for SQLite
│   └── postgres/           # goose migrations for PostgreSQL
│
├── deploy/
│   ├── docker-compose.yml  # Local / self-hosted deployment
│   ├── Dockerfile
│   └── k8s/                # Kubernetes manifests
│
├── docs/
│   ├── openapi.yaml        # API spec (source of truth)
│   ├── architecture/       # ADRs (Architecture Decision Records)
│   └── man/                # Man pages (generated)
│
└── tests/
    ├── unit/               # Pure core logic tests
    ├── integration/        # Database-touching tests
    └── e2e/                # Full stack tests (testcontainers)
```

---

## 7. CLI Design (hledger-style)

```bash
# Core commands
hearth accounts list
hearth accounts add --name "Checking" --type asset --institution "Chase"
hearth transactions add
hearth transactions list --account Checking --since 2025-01-01
hearth transactions import --format ofx ./chase-jan.ofx

# Envelope commands
hearth envelopes list
hearth envelopes allocate --envelope Groceries --amount 600 --period 2025-02

# Reporting
hearth report balance
hearth report income-statement --period 2025-Q1
hearth report net-worth
hearth report budget --period 2025-02

# Server management
hearthd start --config ~/.config/hearth/server.yaml
hearthd status

# Sync
hearth sync --server https://hearth.home.lan

# AI features (requires configured harness)
hearth ai categorize --uncategorized
hearth ai anomalies --since 2025-01-01

# Scriptable output (all commands support these flags)
hearth transactions list --output json
hearth transactions list --output csv
hearth report balance --output json | jq '.accounts[] | select(.type == "asset")'
```

The `--output` flag on every command supports `table` (default, human-readable), `json`, `csv`, and `plain`. This makes every command scriptable without special cases.

---

## 8. Error Design

Errors are a first-class citizen. Every error returned to the user follows this structure:

```
Error: Transaction would violate GAAP balance rule
  Entry total: $150.00 debit, $100.00 credit (difference: $50.00)

  To fix this, you can:
    1. Add a posting to account 'Expenses:Groceries' for $50.00
    2. Adjust an existing posting amount
    3. Run: hearth transactions check --entry <id> for details

  Learn more: hearth help gaap-balance
```

Every error includes:
- **What** happened (machine-readable code)
- **Why** it happened (human context)
- **How** to recover (1-3 concrete options)
- **Where** to learn more (help command or docs link)

---

## 9. Security & Privacy

| Concern | Approach |
|---|---|
| **Auth** | JWT (short-lived, 15min) + refresh tokens (7 days, rotating) |
| **API** | TLS required in server mode; self-signed cert supported for self-hosted |
| **Secrets** | Config via environment variables or OS keychain (never in config files) |
| **SQLite at rest** | Application-level encryption using `go-sqlcipher` or AES-encrypted export |
| **PostgreSQL at rest** | Native PostgreSQL encryption + RLS for household isolation |
| **Logs** | Structured JSON logs (zerolog); no PII, no amounts, no account names in default log level |
| **AI privacy** | Tier 0 default (no calls); Tier 1 private endpoint; Tier 2 external with field-level scrubbing; Tier 2 startup banner not suppressible |
| **Bank feeds** | SimpleFIN Bridge primary connector; credentials handled by bridge, not Hearth; Plaid/Yodlee interfaces stubbed for future use |
| **Multi-tenant** | Row-level security in PostgreSQL; `household_id` on every query |

---

## 10. Multi-User (Household) Model

```
Household "The Smiths"
├── Owner: Robert        (full access: read, write, configure, export)
├── Member: Partner      (read, write transactions; no config access)
└── Viewer: (future)     (read-only; for accountant access, etc.)
```

Permission matrix is enforced at the API middleware layer and validated again in the core (defense in depth). Local mode (SQLite) is always single-user — authentication is not required, but optional PIN/biometric unlock is planned.

---

## 11. Sync Architecture (Local ↔ Server)

The sync problem is hard. The chosen approach is **append-only event log with logical clocks**:

- Every write in local mode appends to a `sync_event` queue with a monotonic sequence number
- On sync, the client sends unsynchronized events; the server applies them in order
- Conflicts (same account edited offline on two devices) are resolved by **last-write-wins at the posting level**, with a **conflict log** written to the database that a human can review
- Journal entries (once posted) are immutable — conflicts only arise on metadata (account names, envelope configurations)

This is intentionally conservative. The goal is never to silently corrupt financial data. Conflicts surface visibly.

---

## 12. Testing Strategy

### Unit Tests (`tests/unit/`)
- All GAAP validation logic tested in pure Go with no database
- Table-driven tests for every error case in the GAAP guard
- Decimal arithmetic edge cases (rounding, currency conversion)
- Target: 90%+ coverage on `internal/core/`

### Integration Tests (`tests/integration/`)
- Store interface tested against both SQLite and PostgreSQL via `testcontainers-go`
- API handlers tested with a real (containerized) database
- Import adapters tested against sample OFX/QIF/CSV files

### End-to-End Tests (`tests/e2e/`)
- Full CLI command execution via `os/exec`
- TUI interaction testing via `teatest` (bubbletea's test harness)
- API tested via generated OpenAPI client

### CI Pipeline
- GitHub Actions: `test`, `lint` (golangci-lint), `build`, `docker`
- `go test -race` enabled for all tests (race condition detection)
- Coverage reports via `go tool cover`

---

## 13. Extensibility Model

The core is intentionally minimal. The extension points are:

| Extension Type | Interface | Example |
|---|---|---|
| Import adapter | `importer.Adapter` | Add Monarch Money CSV format |
| AI provider | `ai.Provider` | Add Anthropic direct integration |
| Report renderer | `report.Renderer` | Add PDF report output |
| Bank connector | `connector.Feed` | Add a new direct-feed institution |
| Export format | `export.Formatter` | Add hledger journal export |

Extensions live in `pkg/` (stable, public interface) and are wired via configuration, not build tags.

---

## 14. Phased Roadmap

### Phase 1 — Core Foundation ✓ **(complete)**
SQLite local mode · Core accounting engine · GAAP guard · CLI (cobra) · goose migrations · Basic account/transaction/journal commands · Unit + integration tests

### Phase 2 — Server Mode
PostgreSQL support · REST API (chi) · JWT auth · Household + member model · RLS · Docker deployment · OpenAPI spec

### Phase 3 — TUI
Bubbletea TUI · Dashboard · Account tree · Transaction entry · Envelope overview

### Phase 4 — Import & Sync
OFX/QIF/CSV importers · SimpleFIN Bridge connector · Local↔Server sync engine · GnuCash XML importer

### Phase 5 — AI Harness
Ollama integration · Transaction categorization · Anomaly detection · Natural language balance queries · Privacy guard

### Phase 6 — iOS Client
Swift/SwiftUI · Consumes REST API · Biometric auth · Transaction entry · Dashboard

### Phase 7 — Reporting & GUI
Toner-friendly printable reports (PDF via `go-wkhtmltopdf` or `fpdf`) · GUI TBD based on Phase 1–6 learnings

---

## 15. Key Dependencies

```
# Core
github.com/shopspring/decimal       # Financial fixed-point arithmetic
github.com/spf13/cobra              # CLI framework
github.com/spf13/viper              # Config management
github.com/charmbracelet/bubbletea  # TUI framework
github.com/charmbracelet/lipgloss   # TUI styling
github.com/charmbracelet/huh        # TUI forms
github.com/go-chi/chi/v5            # HTTP router
github.com/golang-jwt/jwt/v5        # JWT auth

# Database
github.com/jackc/pgx/v5             # PostgreSQL driver
modernc.org/sqlite                  # SQLite (pure Go, no CGO)
github.com/pressly/goose/v3         # Migrations
github.com/sqlc-dev/sqlc            # SQL → Go codegen (dev tool)

# Observability
github.com/rs/zerolog               # Structured logging (no PII)
go.opentelemetry.io/otel            # Tracing (OpenTelemetry)

# Testing
github.com/testcontainers/testcontainers-go  # DB containers for tests
github.com/charmbracelet/teatest    # TUI test harness
github.com/stretchr/testify         # Assertions
```

---

## 16. Architecture Decision Records

ADRs should be written before each phase begins. Initial ADRs to write in `docs/architecture/`:

- `ADR-001-go-as-primary-language.md`
- `ADR-002-event-sourcing-for-ledger.md`
- `ADR-003-sqlite-and-postgres-dual-store.md`
- `ADR-004-openai-compatible-ai-harness.md`
- `ADR-005-jwt-auth-model.md`
- `ADR-007-tiered-ai-privacy-model.md`

ADRs capture *why* a decision was made, not just what was decided. They are invaluable when AI tooling needs context for a non-obvious design choice.

---

*Document version: 0.2 — Phase 1 complete (2026-05-27)*  
*Next step: Phase 2 — Server mode (PostgreSQL, REST API, JWT auth)*
