package currency_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hearth-ledger/hearth/internal/core/currency"
)

func amt(value string, cur currency.Currency) currency.Amount {
	v, err := decimal.NewFromString(value)
	if err != nil {
		panic(err)
	}
	return currency.Amount{Value: v, Currency: cur}
}

func TestAmount_Add_SameCurrency_ReturnsSum(t *testing.T) {
	result, err := amt("10.00", "USD").Add(amt("5.50", "USD"))
	require.NoError(t, err)
	assert.True(t, result.Value.Equal(decimal.RequireFromString("15.50")))
	assert.Equal(t, currency.Currency("USD"), result.Currency)
}

func TestAmount_Add_DifferentCurrencies_ReturnsCurrencyMismatch(t *testing.T) {
	_, err := amt("10.00", "USD").Add(amt("5.00", "EUR"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, currency.ErrCurrencyMismatch))
}

func TestAmount_Sub_SameCurrency_ReturnsDifference(t *testing.T) {
	result, err := amt("10.00", "USD").Sub(amt("3.00", "USD"))
	require.NoError(t, err)
	assert.True(t, result.Value.Equal(decimal.RequireFromString("7.00")))
}

func TestAmount_Sub_DifferentCurrencies_ReturnsCurrencyMismatch(t *testing.T) {
	_, err := amt("10.00", "USD").Sub(amt("3.00", "GBP"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, currency.ErrCurrencyMismatch))
}

func TestAmount_Equal_SameCurrencySameValue_ReturnsTrue(t *testing.T) {
	eq, err := amt("42.00", "USD").Equal(amt("42.00", "USD"))
	require.NoError(t, err)
	assert.True(t, eq)
}

func TestAmount_Equal_SameCurrencyDifferentValue_ReturnsFalse(t *testing.T) {
	eq, err := amt("42.00", "USD").Equal(amt("43.00", "USD"))
	require.NoError(t, err)
	assert.False(t, eq)
}

func TestAmount_Equal_DifferentCurrencies_ReturnsCurrencyMismatch(t *testing.T) {
	_, err := amt("42.00", "USD").Equal(amt("42.00", "EUR"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, currency.ErrCurrencyMismatch))
}

func TestAmount_IsZero_ZeroValue_ReturnsTrue(t *testing.T) {
	assert.True(t, amt("0.00", "USD").IsZero())
}

func TestAmount_IsZero_NonZeroValue_ReturnsFalse(t *testing.T) {
	assert.False(t, amt("0.01", "USD").IsZero())
}

func TestAmount_Neg_ReturnsNegatedValue(t *testing.T) {
	neg := amt("100.00", "USD").Neg()
	assert.True(t, neg.Value.Equal(decimal.RequireFromString("-100.00")))
	assert.Equal(t, currency.Currency("USD"), neg.Currency)
}

func TestAmount_Add_PreservesDecimalPrecision(t *testing.T) {
	// 0.1 + 0.2 must not suffer float rounding
	result, err := amt("0.1", "USD").Add(amt("0.2", "USD"))
	require.NoError(t, err)
	assert.True(t, result.Value.Equal(decimal.RequireFromString("0.3")))
}
