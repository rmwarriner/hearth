package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/member"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// Config holds auth service parameters.
type Config struct {
	JWTSecret       []byte
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	BcryptCost      int
}

// Claims is the JWT payload embedded in every access token.
type Claims struct {
	jwt.RegisteredClaims
	MemberID    string `json:"mid"`
	HouseholdID string `json:"hid"`
	Role        string `json:"role"`
}

// TokenPair is returned by Login and Refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // access token lifetime in seconds
}

// Service handles authentication: login, token refresh, and logout.
type Service struct {
	store  storeapi.Store
	cfg    Config
	logger zerolog.Logger
}

// NewService creates an auth Service. cfg.JWTSecret must be at least 32 bytes.
func NewService(store storeapi.Store, cfg Config, logger zerolog.Logger) (*Service, error) {
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("auth: JWTSecret must be at least 32 bytes")
	}
	if cfg.AccessTokenTTL <= 0 {
		cfg.AccessTokenTTL = 15 * time.Minute
	}
	if cfg.RefreshTokenTTL <= 0 {
		cfg.RefreshTokenTTL = 7 * 24 * time.Hour
	}
	if cfg.BcryptCost <= 0 {
		cfg.BcryptCost = 12
	}
	return &Service{store: store, cfg: cfg, logger: logger}, nil
}

// Login verifies credentials and issues a new token pair with a fresh token family.
func (s *Service) Login(ctx context.Context, householdID account.HouseholdID, email, password string) (TokenPair, error) {
	m, err := s.store.GetMemberByEmail(ctx, householdID, email)
	if err != nil {
		// Return a generic message regardless of whether the account exists.
		return TokenPair{}, hearth.New(hearth.ErrUnauthorized, "invalid credentials").
			WithHints("Check your email and password")
	}

	if !CheckPassword(m.PasswordHash, password) {
		return TokenPair{}, hearth.New(hearth.ErrUnauthorized, "invalid credentials").
			WithHints("Check your email and password")
	}

	familyID, err := randomHex(16)
	if err != nil {
		return TokenPair{}, fmt.Errorf("auth: generate family ID: %w", err)
	}

	pair, err := s.issueTokenPair(ctx, m, familyID)
	if err != nil {
		return TokenPair{}, err
	}

	s.logger.Info().
		Str("member_id", string(m.ID)).
		Str("household_id", string(m.HouseholdID)).
		Str("operation", "login").
		Msg("member authenticated")

	return pair, nil
}

// Refresh validates a refresh token and rotates it: the old token is revoked and
// a new pair is issued within the same family. If the token was already revoked
// (replay attack), the entire family is invalidated and ErrTokenRevoked is returned.
func (s *Service) Refresh(ctx context.Context, rawToken string) (TokenPair, error) {
	hash := HashToken(rawToken)

	rec, err := s.store.GetRefreshToken(ctx, hash)
	if err != nil {
		return TokenPair{}, hearth.New(hearth.ErrTokenInvalid, "refresh token not found").
			WithHints("Log in again to obtain a new token")
	}

	// Replay detection: token was already revoked → invalidate the whole family.
	if rec.RevokedAt != nil {
		if revokeErr := s.store.RevokeRefreshTokenFamily(ctx, rec.FamilyID); revokeErr != nil {
			s.logger.Error().Err(revokeErr).
				Str("family_id", rec.FamilyID).
				Str("operation", "revoke_family_failed").
				Msg("failed to revoke token family after replay detection")
		}
		s.logger.Warn().
			Str("member_id", string(rec.MemberID)).
			Str("family_id", rec.FamilyID).
			Str("operation", "refresh_replay").
			Msg("replay detected — token family revoked")
		return TokenPair{}, hearth.New(hearth.ErrTokenRevoked, "refresh token has already been used").
			WithHints("Log in again — your session has been invalidated for security")
	}

	if time.Now().UTC().After(rec.ExpiresAt) {
		return TokenPair{}, hearth.New(hearth.ErrTokenExpired, "refresh token has expired").
			WithHints("Log in again to obtain a new token")
	}

	// Revoke the old token before issuing the replacement.
	if err := s.store.RevokeRefreshToken(ctx, hash); err != nil {
		return TokenPair{}, fmt.Errorf("auth: revoke old refresh token: %w", err)
	}

	m, err := s.store.GetMember(ctx, rec.MemberID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("auth: load member during refresh: %w", err)
	}

	pair, err := s.issueTokenPair(ctx, m, rec.FamilyID)
	if err != nil {
		return TokenPair{}, err
	}

	s.logger.Info().
		Str("member_id", string(m.ID)).
		Str("operation", "token_refresh").
		Msg("token rotated")

	return pair, nil
}

// Logout revokes the presented refresh token.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	hash := HashToken(rawToken)
	if err := s.store.RevokeRefreshToken(ctx, hash); err != nil {
		return fmt.Errorf("auth: logout: %w", err)
	}
	s.logger.Info().Str("operation", "logout").Msg("refresh token revoked")
	return nil
}

// ValidateAccessToken parses and validates a JWT access token string.
func (s *Service) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.cfg.JWTSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, hearth.New(hearth.ErrTokenExpired, "access token has expired").
				WithHints("Call /auth/refresh to obtain a new token")
		}
		return nil, hearth.New(hearth.ErrTokenInvalid, "invalid access token").
			WithHints("Log in again to obtain a new token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, hearth.New(hearth.ErrTokenInvalid, "malformed token claims")
	}
	return claims, nil
}

// issueTokenPair mints a JWT access token and a random refresh token for m,
// storing the refresh token hash in the store under the given family ID.
func (s *Service) issueTokenPair(ctx context.Context, m member.Member, familyID string) (TokenPair, error) {
	now := time.Now().UTC()

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   string(m.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTokenTTL)),
		},
		MemberID:    string(m.ID),
		HouseholdID: string(m.HouseholdID),
		Role:        string(m.Role),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.cfg.JWTSecret)
	if err != nil {
		return TokenPair{}, fmt.Errorf("auth: sign access token: %w", err)
	}

	// Refresh token: 32-byte random hex string; only its SHA-256 hash is stored.
	// SHA-256 is used (not bcrypt) because the token is already a 32-byte random
	// value — high entropy makes the stored hash safe as a lookup key.
	rawRefresh, err := randomHex(32)
	if err != nil {
		return TokenPair{}, fmt.Errorf("auth: generate refresh token: %w", err)
	}

	if err := s.store.CreateRefreshToken(ctx, storeapi.RefreshToken{
		TokenHash:   HashToken(rawRefresh),
		FamilyID:    familyID,
		MemberID:    m.ID,
		HouseholdID: m.HouseholdID,
		IssuedAt:    now,
		ExpiresAt:   now.Add(s.cfg.RefreshTokenTTL),
	}); err != nil {
		return TokenPair{}, fmt.Errorf("auth: store refresh token: %w", err)
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

// HashNewPassword hashes plain using the service's configured bcrypt cost.
// Used by handlers that create new members or change passwords.
func (s *Service) HashNewPassword(plain string) (string, error) {
	return HashPassword(plain, s.cfg.BcryptCost)
}

// HashToken returns the hex-encoded SHA-256 hash of a raw token string.
// Exported so middleware can hash the Bearer token before store lookup.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// randomHex returns n random bytes encoded as a hex string.
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
