package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/envelope"
)

// CreateEnvelope inserts a new envelope.
func (s *Store) CreateEnvelope(ctx context.Context, e envelope.Envelope) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO envelopes (id, household_id, name, target_amount, target_currency, period_type, rollover_policy)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(e.ID),
		string(e.HouseholdID),
		e.Name,
		e.TargetAmount.Value.String(),
		string(e.TargetAmount.Currency),
		string(e.PeriodType),
		string(e.RolloverPolicy),
	)
	if err != nil {
		return fmt.Errorf("create envelope: %w", toHearthError(err))
	}
	return nil
}

// ListEnvelopes returns all envelopes for a household.
func (s *Store) ListEnvelopes(ctx context.Context, householdID account.HouseholdID) ([]envelope.Envelope, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, household_id, name, target_amount, target_currency, period_type, rollover_policy, created_at
		 FROM envelopes WHERE household_id = ? ORDER BY name ASC`,
		string(householdID),
	)
	if err != nil {
		return nil, fmt.Errorf("list envelopes: %w", err)
	}
	defer rows.Close()

	var envelopes []envelope.Envelope
	for rows.Next() {
		e, err := scanEnvelope(rows)
		if err != nil {
			return nil, fmt.Errorf("scan envelope row: %w", err)
		}
		envelopes = append(envelopes, e)
	}
	return envelopes, rows.Err()
}

// CreateEnvelopeAllocation inserts an allocation record for an envelope.
func (s *Store) CreateEnvelopeAllocation(ctx context.Context, a envelope.Allocation) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO envelope_allocations (id, envelope_id, period_start, amount, currency)
		 VALUES (?, ?, ?, ?, ?)`,
		string(a.ID),
		string(a.EnvelopeID),
		a.PeriodStart.UTC().Format(time.DateOnly),
		a.Amount.Value.String(),
		string(a.Amount.Currency),
	)
	if err != nil {
		return fmt.Errorf("create envelope allocation: %w", toHearthError(err))
	}
	return nil
}

// ListEnvelopeAllocations returns all allocations for an envelope, newest first.
func (s *Store) ListEnvelopeAllocations(ctx context.Context, envelopeID envelope.EnvelopeID) ([]envelope.Allocation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, envelope_id, period_start, amount, currency, created_at
		 FROM envelope_allocations WHERE envelope_id = ? ORDER BY period_start DESC`,
		string(envelopeID),
	)
	if err != nil {
		return nil, fmt.Errorf("list envelope allocations: %w", err)
	}
	defer rows.Close()

	var allocations []envelope.Allocation
	for rows.Next() {
		a, err := scanAllocation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan allocation row: %w", err)
		}
		allocations = append(allocations, a)
	}
	return allocations, rows.Err()
}

type envelopeScanner interface {
	Scan(dest ...any) error
}

func scanEnvelope(sc envelopeScanner) (envelope.Envelope, error) {
	var e envelope.Envelope
	var id, hhID, name, targetAmount, targetCurrency, periodType, rolloverPolicy, createdAt string

	if err := sc.Scan(&id, &hhID, &name, &targetAmount, &targetCurrency, &periodType, &rolloverPolicy, &createdAt); err != nil {
		return envelope.Envelope{}, err
	}

	val, err := decimal.NewFromString(targetAmount)
	if err != nil {
		return envelope.Envelope{}, fmt.Errorf("parse target_amount %q: %w", targetAmount, err)
	}
	t, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		t, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return envelope.Envelope{}, fmt.Errorf("parse created_at %q: %w", createdAt, err)
		}
	}

	e.ID = envelope.EnvelopeID(id)
	e.HouseholdID = account.HouseholdID(hhID)
	e.Name = name
	e.TargetAmount = currency.Amount{Value: val, Currency: currency.Currency(targetCurrency)}
	e.PeriodType = envelope.PeriodType(periodType)
	e.RolloverPolicy = envelope.RolloverPolicy(rolloverPolicy)
	e.CreatedAt = t.UTC()
	return e, nil
}

type allocationScanner interface {
	Scan(dest ...any) error
}

func scanAllocation(sc allocationScanner) (envelope.Allocation, error) {
	var a envelope.Allocation
	var id, envID, periodStart, amount, cur, createdAt string

	if err := sc.Scan(&id, &envID, &periodStart, &amount, &cur, &createdAt); err != nil {
		return envelope.Allocation{}, err
	}

	val, err := decimal.NewFromString(amount)
	if err != nil {
		return envelope.Allocation{}, fmt.Errorf("parse amount %q: %w", amount, err)
	}
	ps, err := time.Parse(time.DateOnly, periodStart)
	if err != nil {
		return envelope.Allocation{}, fmt.Errorf("parse period_start %q: %w", periodStart, err)
	}
	t, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		t, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return envelope.Allocation{}, fmt.Errorf("parse created_at %q: %w", createdAt, err)
		}
	}

	a.ID = envelope.AllocationID(id)
	a.EnvelopeID = envelope.EnvelopeID(envID)
	a.PeriodStart = ps.UTC()
	a.Amount = currency.Amount{Value: val, Currency: currency.Currency(cur)}
	a.CreatedAt = t.UTC()
	return a, nil
}
