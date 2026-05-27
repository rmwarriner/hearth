package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// CreateAccount inserts a new account row.
func (s *Store) CreateAccount(ctx context.Context, a account.Account) error {
	parentID := sql.NullString{}
	if a.ParentID != "" {
		parentID = sql.NullString{String: string(a.ParentID), Valid: true}
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO accounts (id, household_id, name, type, subtype, currency, parent_id, is_placeholder)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		string(a.ID), string(a.HouseholdID), a.Name, string(a.Type),
		a.Subtype, string(a.Currency), parentID,
		boolToInt(a.IsPlaceholder),
	)
	if err != nil {
		return fmt.Errorf("create account: %w", toHearthError(err))
	}
	return nil
}

// GetAccount retrieves a single account by ID.
func (s *Store) GetAccount(ctx context.Context, id account.AccountID) (account.Account, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, household_id, name, type, subtype, currency, parent_id, is_placeholder
		 FROM accounts WHERE id = ?`,
		string(id),
	)

	a, err := scanAccount(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return account.Account{}, hearth.New(hearth.ErrAccountNotFound,
				fmt.Sprintf("account %q not found", id)).
				WithHints("Use `hearth accounts list` to see all accounts").
				WithHelp("accounts")
		}
		return account.Account{}, fmt.Errorf("get account: %w", err)
	}
	return a, nil
}

// ListAccounts returns all accounts belonging to a household.
func (s *Store) ListAccounts(ctx context.Context, householdID account.HouseholdID) ([]account.Account, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, household_id, name, type, subtype, currency, parent_id, is_placeholder
		 FROM accounts WHERE household_id = ? ORDER BY name`,
		string(householdID),
	)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []account.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan account row: %w", err)
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}
	return accounts, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAccount(s scanner) (account.Account, error) {
	var a account.Account
	var id, hhID, name, typ, subtype, cur string
	var parentID sql.NullString
	var isPlaceholder int

	err := s.Scan(&id, &hhID, &name, &typ, &subtype, &cur, &parentID, &isPlaceholder)
	if err != nil {
		return account.Account{}, err
	}

	a.ID = account.AccountID(id)
	a.HouseholdID = account.HouseholdID(hhID)
	a.Name = name
	a.Type = account.AccountType(typ)
	a.Subtype = subtype
	a.Currency = currency.Currency(cur)
	if parentID.Valid {
		a.ParentID = account.AccountID(parentID.String)
	}
	a.IsPlaceholder = isPlaceholder != 0
	return a, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
