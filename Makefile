BINARY_DIR := bin
HEARTH_BIN := $(BINARY_DIR)/hearth
HEARTHD_BIN := $(BINARY_DIR)/hearthd

SQLITE_MIGRATIONS_DIR := migrations/sqlite
TEST_DB := /tmp/hearth-test.db

GO := go
GOFLAGS := -trimpath

.PHONY: all build test lint clean migrate-sqlite

all: build

build: $(HEARTH_BIN) $(HEARTHD_BIN)

$(HEARTH_BIN): $(shell find cmd/hearth internal pkg -name '*.go' 2>/dev/null)
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $@ ./cmd/hearth

$(HEARTHD_BIN): $(shell find cmd/hearthd internal pkg -name '*.go' 2>/dev/null)
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $@ ./cmd/hearthd

test:
	$(GO) test -race -coverprofile=coverage.txt -covermode=atomic ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BINARY_DIR) coverage.txt $(TEST_DB) $(TEST_DB)-wal $(TEST_DB)-shm

migrate-sqlite:
	@echo "Running SQLite migrations against $(TEST_DB)..."
	goose -dir $(SQLITE_MIGRATIONS_DIR) sqlite3 "$(TEST_DB)" up
	@echo "Migrations complete."
