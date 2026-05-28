# Phase 3 — TUI (Terminal UI)

## Overview

Phase 3 delivers an interactive terminal UI for Hearth using the bubbletea/lipgloss/huh stack
from Charm.sh. It runs in **local-only mode** against a direct SQLite connection — the same
store access used by the CLI. Server-connected TUI mode is deferred to Phase 4 (sync).

**Status: Complete** — 2026-05-28

---

## Acceptance Criteria

- [x] `go build ./...` succeeds with zero warnings
- [x] `go test -race ./...` passes on all packages
- [x] `golangci-lint run` reports zero issues
- [x] `hearth tui` launches the full-screen TUI against a local SQLite database
- [x] Four tabs are present: Dashboard, Accounts, Transactions, Envelopes
- [x] Tab navigation works: numeric keys `1`–`4`, `Tab`/`Shift+Tab`
- [x] `[AI: OFF]` indicator is visible in the status bar footer on every screen
- [x] Dashboard shows household name, net worth, and recent transactions
- [x] Accounts screen shows all accounts with current balances
- [x] Accounts screen allows creating a new account via `n` key (huh form)
- [x] Transactions screen shows recent journal entries
- [x] Transactions screen allows creating balanced entries (multi-step form with GAAP validation)
- [x] Envelopes screen shows all envelopes with period/rollover/target
- [x] Envelopes screen allows creating a new envelope via `n` key (huh form)
- [x] Envelope store methods implemented in both SQLite and PostgreSQL
- [x] SQLite integration tests cover all envelope CRUD operations
- [x] TUI unit tests cover: tab bar rendering, AI indicator, tab switching, error overlay, quit

---

## Tasks Completed

| Task | Description |
|---|---|
| 3.1 | ADR-008 (TUI navigation model) + Charm.sh dependencies |
| 3.2 | Envelope store extension (Store interface + SQLite + PostgreSQL + tests) |
| 3.3 | TUI shell: root App model, styles, status bar, error overlay, `hearth tui` command |
| 3.4 | Dashboard screen: async load, spinner, net worth, recent transactions, envelope count |
| 3.5 | Accounts screen: list with balances, create form (huh) |
| 3.6 | Transactions screen: list, multi-step create form, GAAP pre-validation |
| 3.7 | Envelopes screen: list, create form (huh) |
| 3.8 | TUI unit tests using teatest (app model: tab navigation, AI indicator, error overlay) |
| 3.9 | Makefile `tui` target, PHASE3.md, architecture doc update |

---

## Architecture

### Navigation model

```
[1] Dashboard  [2] Accounts  [3] Transactions  [4] Envelopes
```

Keys: `1`–`4` for direct tab jump, `Tab`/`Shift+Tab` to cycle, `q` to quit from any screen.

### Component hierarchy

```
App (root)
├── Chrome: tab bar (top) + status bar (bottom)
│   └── Status bar always shows: [AI: OFF] | Screen name | shortcuts
├── Tab 0 — dashboard.Model
├── Tab 1 — accounts.Model
├── Tab 2 — transactions.Model
└── Tab 3 — envelopes.Model
```

Each screen model is independently testable and has no knowledge of the root App.

### GAAP validation in TUI

The transactions create flow runs `gaap.Validate()` in the review step (Step 3) before
calling `store.CreateJournalEntry`. If violations are found, they are displayed inline
and the user is returned to edit the postings. The store also validates as a safety net.

### Local-only constraint

The TUI calls `internal/store/sqlite` directly — identical to the CLI. It does not
connect to `hearthd`. Server-connected TUI is planned for Phase 4 (sync engine).

---

## File Layout

```
internal/tui/
├── app/
│   ├── app.go          — root model, tab controller, Start(), SetError()
│   └── app_test.go     — teatest unit tests (9 test cases)
├── styles/
│   └── styles.go       — shared lipgloss palette
├── common/
│   ├── statusbar.go    — footer: [AI: OFF] + screen name + shortcuts
│   └── errpanel.go     — dismissable error overlay
├── dashboard/
│   └── dashboard.go    — async load: household, net worth, recent, envelopes
├── accounts/
│   └── accounts.go     — list + huh create form
├── transactions/
│   └── transactions.go — list + multi-step create form + GAAP review
└── envelopes/
    └── envelopes.go    — list + huh create form

internal/store/sqlite/envelope.go   — 4 envelope store methods
internal/store/postgres/envelope.go — 4 envelope store methods
internal/cli/tui_cmd.go             — cobra `hearth tui` subcommand
```

---

## Verification

```bash
# Build both binaries
make build

# Run all tests
make test

# Lint must be clean
make lint

# Launch TUI against a seeded local database
make migrate-sqlite
./bin/hearth init
./bin/hearth tui
```

Manual smoke test:
1. Dashboard shows household name and net worth
2. Press `2` → Accounts list loads
3. Press `n` → account create form appears; fill in and submit → new account in list
4. Press `3` → Transactions list loads
5. Press `n` → multi-step form; enter balanced entry → appears in list
6. Press `n` → enter unbalanced entry → GAAP violations shown inline
7. Press `4` → Envelopes; press `n` → create envelope → appears in list
8. `[AI: OFF]` indicator visible in footer on every screen
9. Press `q` → TUI exits cleanly
