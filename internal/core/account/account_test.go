package account_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hearth-ledger/hearth/internal/core/account"
)

func validAccount() account.Account {
	return account.Account{
		ID:          "acc-uuid-1",
		HouseholdID: "hh-uuid-1",
		Name:        "Checking",
		Type:        account.Asset,
		Currency:    "USD",
	}
}

func TestAccountType_Valid_KnownTypes_ReturnsTrue(t *testing.T) {
	types := []account.AccountType{
		account.Asset,
		account.Liability,
		account.Equity,
		account.Income,
		account.Expense,
	}
	for _, at := range types {
		assert.True(t, at.Valid(), "expected %q to be valid", at)
	}
}

func TestAccountType_Valid_UnknownType_ReturnsFalse(t *testing.T) {
	assert.False(t, account.AccountType("unknown").Valid())
}

func TestAccount_Validate_ValidAccount_ReturnsNil(t *testing.T) {
	err := validAccount().Validate()
	require.NoError(t, err)
}

func TestAccount_Validate_MissingID_ReturnsError(t *testing.T) {
	a := validAccount()
	a.ID = ""
	assert.Error(t, a.Validate())
}

func TestAccount_Validate_MissingHouseholdID_ReturnsError(t *testing.T) {
	a := validAccount()
	a.HouseholdID = ""
	assert.Error(t, a.Validate())
}

func TestAccount_Validate_MissingName_ReturnsError(t *testing.T) {
	a := validAccount()
	a.Name = ""
	assert.Error(t, a.Validate())
}

func TestAccount_Validate_UnknownType_ReturnsError(t *testing.T) {
	a := validAccount()
	a.Type = "bogus"
	assert.Error(t, a.Validate())
}

func TestAccount_Validate_MissingCurrency_ReturnsError(t *testing.T) {
	a := validAccount()
	a.Currency = ""
	assert.Error(t, a.Validate())
}
