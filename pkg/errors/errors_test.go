package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

func TestHearthError_Error_ContainsCodeAndMessage(t *testing.T) {
	e := hearth.New(hearth.ErrGAAPBalance, "postings do not sum to zero")
	assert.Contains(t, e.Error(), "GAAP_BALANCE")
	assert.Contains(t, e.Error(), "postings do not sum to zero")
}

func TestHearthError_UserFacing_MinimalError_ContainsMessage(t *testing.T) {
	e := hearth.New(hearth.ErrAccountNotFound, "account not found")
	out := e.UserFacing()
	assert.Contains(t, out, "Error: account not found")
}

func TestHearthError_UserFacing_WithContext_ContainsContext(t *testing.T) {
	e := hearth.New(hearth.ErrGAAPBalance, "transaction would violate GAAP balance rule").
		WithContext("debits total $150.00 but credits total $100.00")
	out := e.UserFacing()
	assert.Contains(t, out, "debits total")
}

func TestHearthError_UserFacing_WithHints_ContainsNumberedHints(t *testing.T) {
	e := hearth.New(hearth.ErrGAAPBalance, "unbalanced entry").
		WithHints("Add a posting to cover the $50.00 difference", "Adjust an existing posting amount")
	out := e.UserFacing()
	assert.Contains(t, out, "1. Add a posting")
	assert.Contains(t, out, "2. Adjust")
	assert.Contains(t, out, "To fix this, you can:")
}

func TestHearthError_UserFacing_WithHelp_ContainsHelpCommand(t *testing.T) {
	e := hearth.New(hearth.ErrGAAPBalance, "unbalanced entry").
		WithHelp("gaap-balance")
	out := e.UserFacing()
	assert.Contains(t, out, "hearth help gaap-balance")
}

func TestHearthError_UserFacing_FullError_FormatsCorrectly(t *testing.T) {
	e := hearth.New(hearth.ErrGAAPBalance, "transaction would violate GAAP balance rule").
		WithContext("entry total: $150.00 debit, $100.00 credit (difference: $50.00)").
		WithHints(
			"Add a posting to account 'Expenses:Groceries' for $50.00",
			"Adjust an existing posting amount",
		).
		WithHelp("gaap-balance")

	out := e.UserFacing()
	assert.Contains(t, out, "Error: transaction would violate GAAP balance rule")
	assert.Contains(t, out, "entry total:")
	assert.Contains(t, out, "1. Add a posting")
	assert.Contains(t, out, "hearth help gaap-balance")
}

func TestHearthError_Unwrap_ReturnsCause(t *testing.T) {
	cause := errors.New("underlying db error")
	e := hearth.New(hearth.ErrDatabaseConnection, "could not connect").WithCause(cause)
	require.ErrorIs(t, e, cause)
}

func TestHearthError_UserFacing_NoHints_OmitsHintsSection(t *testing.T) {
	e := hearth.New(hearth.ErrAccountNotFound, "account not found")
	out := e.UserFacing()
	assert.NotContains(t, out, "To fix this")
}

func TestHearthError_UserFacing_NoHelp_OmitsLearnMore(t *testing.T) {
	e := hearth.New(hearth.ErrAccountNotFound, "account not found")
	out := e.UserFacing()
	assert.NotContains(t, out, "Learn more")
}
