package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/household"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// CreateHousehold inserts a new household row.
func (s *Store) CreateHousehold(ctx context.Context, h household.Household) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO households (id, name, fiscal_year_start, base_currency)
		 VALUES (?, ?, ?, ?)`,
		string(h.ID), h.Name, h.FiscalYearStart, string(h.BaseCurrency),
	)
	if err != nil {
		return fmt.Errorf("create household: %w", toHearthError(err))
	}
	return nil
}

// GetHousehold retrieves a household by ID.
func (s *Store) GetHousehold(ctx context.Context, id account.HouseholdID) (household.Household, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, fiscal_year_start, base_currency FROM households WHERE id = ?`,
		string(id),
	)

	var h household.Household
	var hID, cur string
	err := row.Scan(&hID, &h.Name, &h.FiscalYearStart, &cur)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return household.Household{}, hearth.New(hearth.ErrHouseholdNotFound,
				fmt.Sprintf("household %q not found", id)).
				WithHints("Run `hearth init` to create a household").
				WithHelp("init")
		}
		return household.Household{}, fmt.Errorf("get household: %w", err)
	}

	h.ID = account.HouseholdID(hID)
	h.BaseCurrency = currency.Currency(cur)
	return h, nil
}
