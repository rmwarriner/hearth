package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
)

// GetAccountBalance computes the balance of an account as of asOf by summing
// all postings up to and including that date.
// Per ADR-004 amounts are stored as TEXT; summation is done in Go, not SQL.
// TODO(phase-7): add a materialized balance cache for report performance.
func (s *Store) GetAccountBalance(ctx context.Context, id account.AccountID, asOf time.Time) (currency.Amount, error) {
	acc, err := s.GetAccount(ctx, id)
	if err != nil {
		return currency.Amount{}, fmt.Errorf("get account for balance: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT p.amount, p.currency
		 FROM postings p
		 JOIN journal_entries je ON je.id = p.journal_entry_id
		 WHERE p.account_id = $1 AND je.posted_at <= $2`,
		string(id),
		asOf.UTC(),
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
			continue // skip foreign-currency postings in Phase 2
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
