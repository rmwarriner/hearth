package account

import (
	"fmt"
	"time"

	"github.com/hearth-ledger/hearth/internal/core/currency"
)

// AccountType classifies an account in the chart of accounts.
type AccountType string

const (
	Asset     AccountType = "asset"
	Liability AccountType = "liability"
	Equity    AccountType = "equity"
	Income    AccountType = "income"
	Expense   AccountType = "expense"
)

// Valid reports whether the account type is one of the recognised values.
func (t AccountType) Valid() bool {
	switch t {
	case Asset, Liability, Equity, Income, Expense:
		return true
	}
	return false
}

// AccountID is a UUID string that uniquely identifies an account.
type AccountID string

// HouseholdID is a UUID string that identifies a household.
type HouseholdID string

// Account is an immutable value representing a ledger account.
// No pointer receivers on value methods — accounts are value types.
type Account struct {
	ID            AccountID         `json:"id"`
	HouseholdID   HouseholdID       `json:"household_id"`
	Name          string            `json:"name"`
	Type          AccountType       `json:"type"`
	Subtype       string            `json:"subtype"`
	Currency      currency.Currency `json:"currency"`
	ParentID      AccountID         `json:"parent_id,omitempty"`
	IsPlaceholder bool              `json:"is_placeholder"`
	CreatedAt     time.Time         `json:"created_at"`
}

// Validate returns an error if the account is structurally invalid.
func (a Account) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("account ID must not be empty")
	}
	if a.HouseholdID == "" {
		return fmt.Errorf("account household ID must not be empty")
	}
	if a.Name == "" {
		return fmt.Errorf("account name must not be empty")
	}
	if !a.Type.Valid() {
		return fmt.Errorf("unknown account type %q", a.Type)
	}
	if a.Currency == "" {
		return fmt.Errorf("account currency must not be empty")
	}
	return nil
}
