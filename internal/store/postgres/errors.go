package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// toHearthError translates PostgreSQL driver errors into typed hearth errors.
func toHearthError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			return hearth.New(hearth.ErrAccountNotFound, "referenced record does not exist").
				WithContext(fmt.Sprintf("A foreign key constraint was violated: %s", pgErr.ConstraintName)).
				WithHints("Ensure all referenced accounts and households exist before creating records.").
				WithCause(err)
		case "23505": // unique_violation
			return hearth.New(hearth.ErrConflict, "record already exists").
				WithContext(fmt.Sprintf("A unique constraint was violated: %s", pgErr.ConstraintName)).
				WithCause(err)
		case "23514": // check_violation
			return hearth.New(hearth.ErrInvalidRequest, "a check constraint was violated").
				WithContext(pgErr.ConstraintName).
				WithCause(err)
		}
	}

	return err
}
