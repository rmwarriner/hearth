# Hearth

A cloud-native household accounting system. Hearth enforces double-entry bookkeeping (GAAP-compliant) with an envelope budgeting view layer on top. It runs entirely offline against a local SQLite database or connects to a self-hosted PostgreSQL server for multi-device, multi-member access. Designed for households, not enterprises.

## Requirements

- Go 1.26+
- [golangci-lint](https://golangci-lint.run/usage/install/) (for `make lint`)
- [goose](https://github.com/pressly/goose) (for `make migrate-sqlite`)
- Docker + Docker Compose (for `hearthd` server mode)

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

## TUI Quick Start

```bash
# After running hearth init, launch the interactive TUI
./bin/hearth tui

# Or via make
make tui
```

## CLI Quick Start (local SQLite)

```bash
# Initialise a local database and create your household
./bin/hearth init

# Add accounts
./bin/hearth accounts add

# Record a transaction
./bin/hearth transactions add

# View balance report
./bin/hearth report balance
```

## Server Mode Quick Start (hearthd + PostgreSQL)

```bash
# Set required secrets
export HEARTH_JWT_SECRET="$(openssl rand -hex 32)"

# Start the server and database
docker compose -f deploy/docker-compose.yml up -d

# Stop and remove volumes
docker compose -f deploy/docker-compose.yml down -v
```

Or run without Docker:

```bash
export HEARTH_DATABASE_URL="postgres://user:pass@localhost:5432/hearth?sslmode=disable"
export HEARTH_JWT_SECRET="your-32-plus-character-secret-here"
./bin/hearthd
```

## Database Setup

```bash
# SQLite (local mode)
make migrate-sqlite

# PostgreSQL (server mode)
HEARTH_TEST_DB_URL="postgres://..." make migrate-postgres
```

## Architecture

See [`docs/architecture/hearth-architecture.md`](docs/architecture/hearth-architecture.md) for the full design document and [`docs/architecture/adr/`](docs/architecture/adr/) for Architecture Decision Records.

**Current status:** Phase 3 (TUI) complete. See [`docs/architecture/PHASE3.md`](docs/architecture/PHASE3.md) for the Phase 3 spec and acceptance criteria.
