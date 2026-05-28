package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/member"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// CreateMember inserts a new member row.
func (s *Store) CreateMember(ctx context.Context, m member.Member) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := SetHouseholdContext(ctx, tx, m.HouseholdID); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO members (id, household_id, display_name, email, role, password_hash)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		string(m.ID), string(m.HouseholdID), m.DisplayName, m.Email,
		string(m.Role), m.PasswordHash,
	)
	if err != nil {
		return fmt.Errorf("create member: %w", toHearthError(err))
	}
	return tx.Commit()
}

// GetMember retrieves a member by ID.
func (s *Store) GetMember(ctx context.Context, id member.MemberID) (member.Member, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, household_id, display_name, email, role, password_hash, created_at
		 FROM members WHERE id = $1`,
		string(id),
	)
	m, err := scanMember(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return member.Member{}, hearth.New(hearth.ErrMemberNotFound,
				fmt.Sprintf("member %q not found", id)).
				WithHints("Verify the member ID is correct")
		}
		return member.Member{}, fmt.Errorf("get member: %w", err)
	}
	return m, nil
}

// GetMemberByEmail retrieves a member by email within a household.
func (s *Store) GetMemberByEmail(ctx context.Context, householdID account.HouseholdID, email string) (member.Member, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, household_id, display_name, email, role, password_hash, created_at
		 FROM members WHERE household_id = $1 AND email = $2`,
		string(householdID), email,
	)
	m, err := scanMember(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return member.Member{}, hearth.New(hearth.ErrMemberNotFound,
				fmt.Sprintf("no member with email %q in this household", email)).
				WithHints("Check the email address")
		}
		return member.Member{}, fmt.Errorf("get member by email: %w", err)
	}
	return m, nil
}

// ListMembers returns all members belonging to a household.
func (s *Store) ListMembers(ctx context.Context, householdID account.HouseholdID) ([]member.Member, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, household_id, display_name, email, role, password_hash, created_at
		 FROM members WHERE household_id = $1 ORDER BY created_at`,
		string(householdID),
	)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []member.Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, fmt.Errorf("scan member row: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return members, nil
}

// UpdateMemberRole changes the role of an existing member.
func (s *Store) UpdateMemberRole(ctx context.Context, id member.MemberID, role member.Role) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE members SET role = $1 WHERE id = $2`,
		string(role), string(id),
	)
	if err != nil {
		return fmt.Errorf("update member role: %w", toHearthError(err))
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update member role rows affected: %w", err)
	}
	if n == 0 {
		return hearth.New(hearth.ErrMemberNotFound,
			fmt.Sprintf("member %q not found", id)).
			WithHints("Verify the member ID is correct")
	}
	return nil
}

type memberScanner interface {
	Scan(dest ...any) error
}

func scanMember(s memberScanner) (member.Member, error) {
	var m member.Member
	var id, householdID, role string
	var createdAt time.Time

	err := s.Scan(&id, &householdID, &m.DisplayName, &m.Email, &role, &m.PasswordHash, &createdAt)
	if err != nil {
		return member.Member{}, err
	}

	m.ID = member.MemberID(id)
	m.HouseholdID = account.HouseholdID(householdID)
	m.Role = member.Role(role)
	m.CreatedAt = createdAt
	return m, nil
}
