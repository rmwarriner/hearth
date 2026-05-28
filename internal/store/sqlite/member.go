package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/member"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// CreateMember inserts a new member row.
func (s *Store) CreateMember(ctx context.Context, m member.Member) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO members (id, household_id, display_name, email, role, password_hash)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		string(m.ID), string(m.HouseholdID), m.DisplayName, m.Email,
		string(m.Role), m.PasswordHash,
	)
	if err != nil {
		return fmt.Errorf("create member: %w", toHearthError(err))
	}
	return nil
}

// GetMember retrieves a member by ID.
func (s *Store) GetMember(ctx context.Context, id member.MemberID) (member.Member, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, household_id, display_name, email, role, password_hash, created_at
		 FROM members WHERE id = ?`,
		string(id),
	)
	m, err := scanMember(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return member.Member{}, hearth.New(hearth.ErrMemberNotFound,
				fmt.Sprintf("member %q not found", id)).
				WithHints("Verify the member ID is correct").
				WithHelp("members")
		}
		return member.Member{}, fmt.Errorf("get member: %w", err)
	}
	return m, nil
}

// GetMemberByEmail retrieves a member by email within a household.
func (s *Store) GetMemberByEmail(ctx context.Context, householdID account.HouseholdID, email string) (member.Member, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, household_id, display_name, email, role, password_hash, created_at
		 FROM members WHERE household_id = ? AND email = ?`,
		string(householdID), email,
	)
	m, err := scanMember(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return member.Member{}, hearth.New(hearth.ErrMemberNotFound,
				fmt.Sprintf("no member with email %q in this household", email)).
				WithHints("Check the email address", "Run `hearth members list` to see all members").
				WithHelp("members")
		}
		return member.Member{}, fmt.Errorf("get member by email: %w", err)
	}
	return m, nil
}

// ListMembers returns all members belonging to a household.
func (s *Store) ListMembers(ctx context.Context, householdID account.HouseholdID) ([]member.Member, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, household_id, display_name, email, role, password_hash, created_at
		 FROM members WHERE household_id = ? ORDER BY created_at`,
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
			return nil, fmt.Errorf("scan member: %w", err)
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
		`UPDATE members SET role = ? WHERE id = ?`,
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

// CreateRefreshToken is not supported in SQLite (local/single-user mode).
func (s *Store) CreateRefreshToken(_ context.Context, _ storeapi.RefreshToken) error {
	return hearth.New(hearth.ErrNotImplemented, "refresh tokens require server mode").
		WithContext("The SQLite store is used in local mode; refresh tokens are only available when running hearthd with PostgreSQL.").
		WithHints("Start hearthd with a PostgreSQL database URL to enable authentication")
}

// GetRefreshToken is not supported in SQLite (local/single-user mode).
func (s *Store) GetRefreshToken(_ context.Context, _ string) (storeapi.RefreshToken, error) {
	return storeapi.RefreshToken{}, hearth.New(hearth.ErrNotImplemented, "refresh tokens require server mode")
}

// RevokeRefreshToken is not supported in SQLite (local/single-user mode).
func (s *Store) RevokeRefreshToken(_ context.Context, _ string) error {
	return hearth.New(hearth.ErrNotImplemented, "refresh tokens require server mode")
}

// RevokeRefreshTokenFamily is not supported in SQLite (local/single-user mode).
func (s *Store) RevokeRefreshTokenFamily(_ context.Context, _ string) error {
	return hearth.New(hearth.ErrNotImplemented, "refresh tokens require server mode")
}

func scanMember(s scanner) (member.Member, error) {
	var m member.Member
	var id, householdID, role string
	var createdAt string

	err := s.Scan(&id, &householdID, &m.DisplayName, &m.Email, &role, &m.PasswordHash, &createdAt)
	if err != nil {
		return member.Member{}, err
	}

	m.ID = member.MemberID(id)
	m.HouseholdID = account.HouseholdID(householdID)
	m.Role = member.Role(role)

	t, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		// Fall back to date-only format.
		t, err = time.Parse(time.DateOnly, createdAt)
		if err != nil {
			return member.Member{}, fmt.Errorf("parse created_at %q: %w", createdAt, err)
		}
	}
	m.CreatedAt = t
	return m, nil
}
