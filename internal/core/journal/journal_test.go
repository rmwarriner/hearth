package journal_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/journal"
)

func TestJournalEntry_SourceConstants_AreDistinct(t *testing.T) {
	assert.NotEqual(t, journal.SourceManual, journal.SourceImport)
	assert.NotEqual(t, journal.SourceManual, journal.SourceReversal)
}

func TestPosting_Amount_IsAmountType_NotRawDecimal(t *testing.T) {
	p := journal.Posting{
		Amount: currency.Amount{
			Value:    decimal.RequireFromString("100.00"),
			Currency: "USD",
		},
	}
	assert.Equal(t, currency.Currency("USD"), p.Amount.Currency)
}
