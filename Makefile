BINARY_DIR := bin
HEARTH_BIN := $(BINARY_DIR)/hearth
HEARTHD_BIN := $(BINARY_DIR)/hearthd

SQLITE_MIGRATIONS_DIR := migrations/sqlite
POSTGRES_MIGRATIONS_DIR := migrations/postgres
TEST_DB := /tmp/hearth-test.db
HEARTH_TEST_DB_URL ?= postgres://hearth:hearth@localhost:5432/hearth?sslmode=disable

OPENAPI_SPEC := docs/openapi.yaml
OPENAPI_OUT  := internal/api/openapi/api.gen.go
OAPI_CODEGEN := $(shell go env GOPATH)/bin/oapi-codegen

GO := go
GOFLAGS := -trimpath

DOCKER_COMPOSE := docker compose -f deploy/docker-compose.yml

.PHONY: all build test lint clean migrate-sqlite migrate-postgres generate-api \
        docker-build docker-up docker-down test-integration-postgres

all: build

build: $(HEARTH_BIN) $(HEARTHD_BIN)

$(HEARTH_BIN): $(shell find cmd/hearth internal pkg -name '*.go' 2>/dev/null)
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $@ ./cmd/hearth

$(HEARTHD_BIN): $(shell find cmd/hearthd internal pkg -name '*.go' 2>/dev/null)
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $@ ./cmd/hearthd

test:
ifeq ($(HEARTH_SKIP_INTEGRATION),1)
	$(GO) test -race -coverprofile=coverage.txt -covermode=atomic \
		$(shell go list ./... | grep -v tests/integration)
else
	$(GO) test -race -coverprofile=coverage.txt -covermode=atomic ./...
endif

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BINARY_DIR) coverage.txt $(TEST_DB) $(TEST_DB)-wal $(TEST_DB)-shm

migrate-sqlite:
	@echo "Running SQLite migrations against $(TEST_DB)..."
	goose -dir $(SQLITE_MIGRATIONS_DIR) sqlite3 "$(TEST_DB)" up
	@echo "Migrations complete."

migrate-postgres:
	@echo "Running PostgreSQL migrations against $(HEARTH_TEST_DB_URL)..."
	goose -dir $(POSTGRES_MIGRATIONS_DIR) postgres "$(HEARTH_TEST_DB_URL)" up
	@echo "Migrations complete."

generate-api:
	@which $(OAPI_CODEGEN) > /dev/null 2>&1 || go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	$(OAPI_CODEGEN) -generate types,chi-server -package openapi \
		-o $(OPENAPI_OUT) $(OPENAPI_SPEC)
	@echo "Generated $(OPENAPI_OUT)"

docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down -v

test-integration-postgres:
	HEARTH_TEST_DB_URL=$(HEARTH_TEST_DB_URL) \
		$(GO) test -race -v -timeout 120s ./tests/integration/...
