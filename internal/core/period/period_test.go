package period_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hearth-ledger/hearth/internal/core/period"
)

func TestFiscalPeriod_IsLocked_NilLockedAt_ReturnsFalse(t *testing.T) {
	p := period.FiscalPeriod{
		ID:        "p-1",
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, 1, 0),
		LockedAt:  nil,
	}
	assert.False(t, p.IsLocked())
}

func TestFiscalPeriod_IsLocked_NonNilLockedAt_ReturnsTrue(t *testing.T) {
	now := time.Now()
	p := period.FiscalPeriod{
		ID:        "p-2",
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, 1, 0),
		LockedAt:  &now,
	}
	assert.True(t, p.IsLocked())
}
