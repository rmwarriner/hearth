package postgres

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

// CreateRefreshToken stores a hashed refresh token.
func (s *Store) CreateRefreshToken(ctx context.Context, t storeapi.RefreshToken) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (token_hash, family_id, member_id, household_id, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.TokenHash, t.FamilyID, string(t.MemberID), string(t.HouseholdID), t.ExpiresAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", toHearthError(err))
	}
	return nil
}

// GetRefreshToken retrieves a refresh token record by its hash.
func (s *Store) GetRefreshToken(ctx context.Context, tokenHash string) (storeapi.RefreshToken, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT token_hash, family_id, member_id, household_id, issued_at, expires_at, revoked_at
		 FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	)

	var t storeapi.RefreshToken
	var memberID, householdID string
	var revokedAt sql.NullTime

	err := row.Scan(
		&t.TokenHash, &t.FamilyID, &memberID, &householdID,
		&t.IssuedAt, &t.ExpiresAt, &revokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storeapi.RefreshToken{}, hearth.New(hearth.ErrTokenInvalid, "refresh token not found")
		}
		return storeapi.RefreshToken{}, fmt.Errorf("get refresh token: %w", err)
	}

	t.MemberID = member.MemberID(memberID)
	t.HouseholdID = account.HouseholdID(householdID)
	t.IssuedAt = t.IssuedAt.UTC()
	t.ExpiresAt = t.ExpiresAt.UTC()
	if revokedAt.Valid {
		rev := revokedAt.Time.UTC()
		t.RevokedAt = &rev
	}
	return t, nil
}

// RevokeRefreshToken marks a single refresh token as revoked.
func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE token_hash = $2 AND revoked_at IS NULL`,
		now, tokenHash,
	)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke refresh token rows affected: %w", err)
	}
	if n == 0 {
		return hearth.New(hearth.ErrTokenInvalid, "token not found or already revoked")
	}
	return nil
}

// RevokeRefreshTokenFamily revokes all tokens belonging to the same family.
// Called on replay detection — if a revoked token is presented, the entire
// login chain is invalidated and the member must re-authenticate.
func (s *Store) RevokeRefreshTokenFamily(ctx context.Context, familyID string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE family_id = $2 AND revoked_at IS NULL`,
		now, familyID,
	)
	if err != nil {
		return fmt.Errorf("revoke refresh token family: %w", err)
	}
	return nil
}
