package postgres

import (
	"database/sql"

	storeapi "github.com/hearth-ledger/hearth/internal/store"
)

// Store is the PostgreSQL implementation of store.Store.
type Store struct {
	db *sql.DB
}

// New creates a Store backed by the given *sql.DB (obtained from postgres.Open).
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// compile-time assertion that Store satisfies the store.Store interface.
var _ storeapi.Store = (*Store)(nil)
