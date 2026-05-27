package sqlite

import (
	"database/sql"

	storeapi "github.com/hearth-ledger/hearth/internal/store"
)

// compile-time assertion that Store satisfies the store.Store interface.
var _ storeapi.Store = (*Store)(nil)

// Store is the SQLite implementation of store.Store.
type Store struct {
	db *sql.DB
}

// New returns a Store backed by an already-opened SQLite database.
// Use Open to open and migrate the database before calling New.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}
