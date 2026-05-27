package household

import (
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
)

// Household is the top-level tenant in Hearth. All data belongs to a household.
type Household struct {
	ID              account.HouseholdID
	Name            string
	FiscalYearStart int // 1-based month (1 = January)
	BaseCurrency    currency.Currency
	CreatedAt       time.Time
}
