package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/hearth-ledger/hearth/internal/core/account"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Config holds the parameters needed to open a PostgreSQL connection pool.
type Config struct {
	DSN            string
	MaxConns       int32
	MinConns       int32
	ConnectTimeout time.Duration
}

// Open creates a connection pool, validates it, and runs any pending migrations.
// The returned *sql.DB is safe for concurrent use.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	if cfg.MaxConns < 1 {
		return nil, fmt.Errorf("postgres: MaxConns must be >= 1, got %d", cfg.MaxConns)
	}
	if cfg.ConnectTimeout <= 0 {
		return nil, fmt.Errorf("postgres: ConnectTimeout must be > 0")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse DSN: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres: migrations: %w", err)
	}

	return db, nil
}

// runMigrations runs all pending goose migrations from the embedded FS.
func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

// SetHouseholdContext sets the transaction-local `app.household_id` PostgreSQL
// setting so that RLS policies can enforce household isolation.
// The `true` flag passed to set_config makes the setting transaction-local —
// it resets automatically on COMMIT or ROLLBACK.
func SetHouseholdContext(ctx context.Context, tx *sql.Tx, householdID account.HouseholdID) error {
	_, err := tx.ExecContext(ctx,
		`SELECT set_config('app.household_id', $1, true)`,
		string(householdID),
	)
	if err != nil {
		return fmt.Errorf("set household context: %w", err)
	}
	return nil
}
