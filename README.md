# Hearth

A cloud-native household accounting system. Hearth enforces double-entry bookkeeping (GAAP-compliant) with an envelope budgeting view layer on top. It runs entirely offline against a local SQLite database or connects to a self-hosted PostgreSQL server for multi-device sync. Designed for households, not enterprises.

## Requirements

- Go 1.23+
- [golangci-lint](https://golangci-lint.run/usage/install/) (for `make lint`)
- [goose](https://github.com/pressly/goose) (for `make migrate-sqlite`)

## Build

```bash
# Install dependencies
go mod download

# Build both binaries
make build

# Binaries land in bin/
./bin/hearth version
```

## Test

```bash
make test
```

## Lint

```bash
make lint
```

## Database setup (local SQLite)

```bash
# Run all migrations against a local test database
make migrate-sqlite

# Or use the CLI to initialize your personal database
./bin/hearth init
```

## Quick start

```bash
./bin/hearth init
./bin/hearth accounts add
./bin/hearth transactions add
./bin/hearth report balance
```

## Architecture

See [`docs/architecture/hearth-architecture.md`](docs/architecture/hearth-architecture.md) for the full design document and [`docs/architecture/adr/`](docs/architecture/adr/) for Architecture Decision Records.

**Current status:** Phase 1 (Core Foundation) complete. See [`docs/architecture/PHASE1.md`](docs/architecture/PHASE1.md) for the Phase 1 spec and acceptance criteria.
