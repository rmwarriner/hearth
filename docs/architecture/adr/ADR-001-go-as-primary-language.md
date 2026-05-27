# ADR-001: Go as Primary Language

## Status
Accepted

## Context
Hearth needs a single static binary that works offline on macOS and Linux, a first-class CLI/TUI experience, correct fixed-point decimal arithmetic, and clean AI-assisted code generation. The Charm.sh TUI ecosystem (bubbletea, lipgloss, huh) and the cobra/viper CLI ecosystem are both Go-native and purpose-built for this use case. The alternative evaluated was Rust (stronger type safety, faster binaries) and TypeScript/Node (wider library ecosystem). Neither has an equivalent to the Charm.sh ecosystem, and both require more complex packaging for single-binary distribution.

## Decision
Go 1.23+ is the primary implementation language. Both CLI (`hearth`) and server daemon (`hearthd`) are Go binaries compiled from the same module.

## Consequences
- The `shopspring/decimal` library handles all monetary arithmetic; floating-point types are banned from financial fields.
- `modernc.org/sqlite` (pure Go, CGO-free) is the SQLite driver, which preserves the single-binary property.
- `sqlc` generates type-safe Go from raw SQL, keeping the data layer readable and reviewable.
- Cross-compilation to `darwin/arm64`, `darwin/amd64`, `linux/amd64`, and `linux/arm64` is straightforward.
- The iOS client will be a separate Swift/SwiftUI codebase consuming the REST API; no shared Go/Swift code is planned.
