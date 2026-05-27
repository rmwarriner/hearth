package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/period"
)

// CreateFiscalPeriod inserts a new fiscal period.
func (s *Store) CreateFiscalPeriod(ctx context.Context, p period.FiscalPeriod) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO fiscal_periods (id, household_id, name, start_date, end_date, locked_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		string(p.ID),
		string(p.HouseholdID),
		p.Name,
		p.StartDate.UTC().Format(time.DateOnly),
		p.EndDate.UTC().Format(time.DateOnly),
		nullableTime(p.LockedAt),
	)
	if err != nil {
		return fmt.Errorf("create fiscal period: %w", toHearthError(err))
	}
	return nil
}

// LockFiscalPeriod sets the locked_at timestamp for a fiscal period.
func (s *Store) LockFiscalPeriod(ctx context.Context, id period.PeriodID) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx,
		`UPDATE fiscal_periods SET locked_at = ? WHERE id = ? AND locked_at IS NULL`,
		now, string(id),
	)
	if err != nil {
		return fmt.Errorf("lock fiscal period: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("lock fiscal period rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("fiscal period %q not found or already locked", id)
	}
	return nil
}

// ListLockedPeriods returns all locked periods for a household, used by the GAAP guard.
func (s *Store) ListLockedPeriods(ctx context.Context, householdID account.HouseholdID) ([]period.FiscalPeriod, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, household_id, name, start_date, end_date, locked_at
		 FROM fiscal_periods WHERE household_id = ? AND locked_at IS NOT NULL`,
		string(householdID),
	)
	if err != nil {
		return nil, fmt.Errorf("list locked periods: %w", err)
	}
	defer rows.Close()

	var periods []period.FiscalPeriod
	for rows.Next() {
		p, err := scanPeriod(rows)
		if err != nil {
			return nil, fmt.Errorf("scan period row: %w", err)
		}
		periods = append(periods, p)
	}
	return periods, rows.Err()
}

type periodScanner interface {
	Scan(dest ...any) error
}

func scanPeriod(sc periodScanner) (period.FiscalPeriod, error) {
	var p period.FiscalPeriod
	var id, hhID, name, start, end string
	var lockedAt sql.NullString

	if err := sc.Scan(&id, &hhID, &name, &start, &end, &lockedAt); err != nil {
		return period.FiscalPeriod{}, err
	}

	startDate, err := time.Parse(time.DateOnly, start)
	if err != nil {
		return period.FiscalPeriod{}, fmt.Errorf("parse start_date: %w", err)
	}
	endDate, err := time.Parse(time.DateOnly, end)
	if err != nil {
		return period.FiscalPeriod{}, fmt.Errorf("parse end_date: %w", err)
	}

	p.ID = period.PeriodID(id)
	p.HouseholdID = account.HouseholdID(hhID)
	p.Name = name
	p.StartDate = startDate
	p.EndDate = endDate

	if lockedAt.Valid {
		t, err := time.Parse(time.RFC3339Nano, lockedAt.String)
		if err != nil {
			return period.FiscalPeriod{}, fmt.Errorf("parse locked_at: %w", err)
		}
		p.LockedAt = &t
	}
	return p, nil
}

func nullableTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(time.RFC3339Nano), Valid: true}
}
