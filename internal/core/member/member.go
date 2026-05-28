package member

import (
	"fmt"
	"strings"
	"time"

	// TODO(phase-3): consider moving HouseholdID to internal/core/household
	"github.com/hearth-ledger/hearth/internal/core/account"
)

// MemberID is a UUID string identifying a household member.
type MemberID string

// Role controls what a member is permitted to do within a household.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// Valid reports whether r is a recognised role.
func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleMember, RoleViewer:
		return true
	}
	return false
}

// Member represents a person who belongs to a household.
// PasswordHash is an opaque bcrypt digest — hashing and verification live in
// the auth service layer, not here.
type Member struct {
	ID           MemberID
	HouseholdID  account.HouseholdID
	DisplayName  string
	Email        string
	Role         Role
	PasswordHash string
	CreatedAt    time.Time
}

// Validate checks that the Member has all required fields set correctly.
func (m Member) Validate() error {
	var errs []string

	if m.ID == "" {
		errs = append(errs, "ID is required")
	}
	if m.HouseholdID == "" {
		errs = append(errs, "HouseholdID is required")
	}
	if m.DisplayName == "" {
		errs = append(errs, "DisplayName is required")
	}
	if m.Email == "" {
		errs = append(errs, "Email is required")
	} else if !strings.Contains(m.Email, "@") {
		errs = append(errs, "Email must contain @")
	}
	if !m.Role.Valid() {
		errs = append(errs, fmt.Sprintf("Role %q is not valid (must be owner, member, or viewer)", m.Role))
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid member: %s", strings.Join(errs, "; "))
	}
	return nil
}
