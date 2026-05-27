package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// GetAccountBalance computes the balance of an account as of asOf by summing
// all postings up to and including that date. The currency is taken from the
// account definition; multi-currency accounts are not supported in Phase 1.
func (s *Store) GetAccountBalance(ctx context.Context, id account.AccountID, asOf time.Time) (currency.Amount, error) {
	// Retrieve the account's currency.
	acc, err := s.GetAccount(ctx, id)
	if err != nil {
		return currency.Amount{}, fmt.Errorf("get account for balance: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT p.amount, p.currency
		 FROM postings p
		 JOIN journal_entries je ON je.id = p.journal_entry_id
		 WHERE p.account_id = ? AND je.posted_at <= ?`,
		string(id),
		asOf.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return currency.Amount{}, fmt.Errorf("query postings for balance: %w", err)
	}
	defer rows.Close()

	total := decimal.Zero
	for rows.Next() {
		var amountStr, cur string
		if err := rows.Scan(&amountStr, &cur); err != nil {
			return currency.Amount{}, fmt.Errorf("scan posting: %w", err)
		}
		if currency.Currency(cur) != acc.Currency {
			continue // skip foreign-currency postings in Phase 1
		}
		val, err := decimal.NewFromString(amountStr)
		if err != nil {
			return currency.Amount{}, fmt.Errorf("parse amount %q: %w", amountStr, err)
		}
		total = total.Add(val)
	}
	if err := rows.Err(); err != nil {
		return currency.Amount{}, fmt.Errorf("iterate postings: %w", err)
	}

	return currency.Amount{Value: total, Currency: acc.Currency}, nil
}

// toHearthError translates database errors to typed hearth errors where possible.
func toHearthError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return hearth.New(hearth.ErrAccountNotFound, "record not found").WithCause(err)
	}
	// SQLite error strings contain "FOREIGN KEY constraint failed" for FK violations.
	errStr := err.Error()
	if contains(errStr, "FOREIGN KEY constraint failed") {
		return hearth.New(hearth.ErrAccountNotFound, "referenced record does not exist").
			WithContext("A foreign key constraint was violated.").
			WithHints("Ensure all referenced accounts and households exist before creating records.").
			WithCause(err)
	}
	if contains(errStr, "UNIQUE constraint failed") {
		return hearth.New(hearth.ErrInternal, "record already exists").WithCause(err)
	}
	return err
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && indexOfString(s, sub) >= 0)
}

func indexOfString(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
