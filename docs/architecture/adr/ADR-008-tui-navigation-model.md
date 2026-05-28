# ADR-008: TUI Screen Architecture and Navigation Model

## Status
Accepted

## Context
Phase 3 adds an interactive terminal UI (`hearth tui`) using the Charm.sh bubbletea/lipgloss/huh stack. Several architectural decisions needed to be made before implementation:

1. **How is the application structured?** Bubbletea's functional update model could be implemented as a single monolithic model or as a composed hierarchy of models.
2. **How does the user navigate between screens?** Options include a top-level menu list, tab bar, command palette, or numbered shortcuts.
3. **Where does the AI tier indicator live?** CLAUDE.md requires it on every screen. It could be per-screen or in a shared chrome layer.
4. **What library handles forms?** The CLAUDE.md approved stack includes `charmbracelet/huh`.
5. **Is Phase 3 TUI local-only or server-connected?** The architecture roadmap explicitly places server sync in Phase 4.

## Decision

### Composition pattern
The TUI uses a **root `App` model** that owns the global chrome (tab bar, status bar, error overlay) and delegates all screen-specific logic to four child `tea.Model` instances — one per tab. The `App` model dispatches `tea.Msg` to the active child and re-renders the child's `View()` within the chrome.

This pattern keeps each screen independently testable (the child models have no knowledge of the tab bar or status bar), and makes the common chrome (including the AI tier indicator) a single implementation rather than one duplicated in each screen.

### Navigation model
A **tab bar** with four named tabs, selectable by:
- Numeric keys `1`–`4`
- `Tab` to advance, `Shift+Tab` to go back
- `q` / `Ctrl+C` to quit from any screen

Tabs: `[1] Dashboard  [2] Accounts  [3] Transactions  [4] Envelopes`

A tab bar was chosen over a menu list because: the number of screens is small and fixed (4); numbered shortcuts allow instant one-key navigation; the active tab label provides constant orientation without a separate breadcrumb.

### AI tier indicator
The indicator lives in the **status bar footer**, rendered by the root `App` model on every `View()` call, regardless of which tab is active. Format: `[AI: OFF]` in Phase 3 (no AI implemented). The footer also shows the current screen name (centre) and shortcut hints (right).

This placement satisfies CLAUDE.md's "persistent status bar" requirement and ensures the indicator is never absent even during loading states or form flows.

### Form library
`charmbracelet/huh` for all create/edit forms. It is purpose-built for bubbletea, handles focus management and validation feedback natively, and is on the approved CLAUDE.md tech stack.

### Local mode only
Phase 3 TUI operates **exclusively in local SQLite mode** — the same direct store access as the CLI. It does not connect to `hearthd` over HTTP. Server-connected TUI requires the sync engine planned for Phase 4.

### Test harness
`github.com/charmbracelet/x/exp/teatest` (the Charm.sh experimental teatest package) is used for TUI interaction tests. It provides `NewTestModel` (unit-level model testing) and `NewTestProgram` (full program e2e testing with programmatic keypress injection and `View()` snapshot assertions).

## Consequences

- Each screen model (`dashboard`, `accounts`, `transactions`, `envelopes`) has no dependency on the root `App` — clean separation, easy to test in isolation.
- Global key handling (`q`, `1`–`4`, `Tab`) lives only in `App.Update`, preventing key conflicts between the chrome and screen-level key bindings.
- The status bar is always present, so the AI tier indicator is guaranteed visible even when screens are loading or showing modal forms.
- Adding a fifth screen in a later phase requires adding one entry to the tab bar and one child model in `App` — no structural changes.
- Local-only constraint means users of `hearthd` do not get a TUI in Phase 3. This is intentional and documented; Phase 4 will bridge the gap.
