package currency

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Currency is an ISO 4217 currency code (e.g. "USD", "EUR").
type Currency string

// Amount is a monetary value with an explicit currency.
// Never use decimal.Decimal directly for money — always use Amount.
type Amount struct {
	Value    decimal.Decimal
	Currency Currency
}

// Add returns the sum of two amounts. Both must share the same currency.
func (a Amount) Add(b Amount) (Amount, error) {
	if a.Currency != b.Currency {
		return Amount{}, fmt.Errorf("cannot add %s and %s: %w", a.Currency, b.Currency, ErrCurrencyMismatch)
	}
	return Amount{Value: a.Value.Add(b.Value), Currency: a.Currency}, nil
}

// Sub returns a minus b. Both must share the same currency.
func (a Amount) Sub(b Amount) (Amount, error) {
	if a.Currency != b.Currency {
		return Amount{}, fmt.Errorf("cannot subtract %s from %s: %w", b.Currency, a.Currency, ErrCurrencyMismatch)
	}
	return Amount{Value: a.Value.Sub(b.Value), Currency: a.Currency}, nil
}

// Equal reports whether two amounts are numerically equal and share a currency.
func (a Amount) Equal(b Amount) (bool, error) {
	if a.Currency != b.Currency {
		return false, fmt.Errorf("cannot compare %s and %s: %w", a.Currency, b.Currency, ErrCurrencyMismatch)
	}
	return a.Value.Equal(b.Value), nil
}

// IsZero reports whether the amount value is zero.
func (a Amount) IsZero() bool {
	return a.Value.IsZero()
}

// Neg returns the negation of the amount.
func (a Amount) Neg() Amount {
	return Amount{Value: a.Value.Neg(), Currency: a.Currency}
}

// String returns a human-readable representation.
func (a Amount) String() string {
	return a.Value.String() + " " + string(a.Currency)
}

// ErrCurrencyMismatch is returned when an operation is attempted on amounts
// with different currencies.
var ErrCurrencyMismatch = fmt.Errorf("currency mismatch")
