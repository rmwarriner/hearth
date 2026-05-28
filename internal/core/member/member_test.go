package member_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/member"
)

func validMember() member.Member {
	return member.Member{
		ID:           member.MemberID("m-1"),
		HouseholdID:  account.HouseholdID("hh-1"),
		DisplayName:  "Alice",
		Email:        "alice@example.com",
		Role:         member.RoleOwner,
		PasswordHash: "$2a$12$fakehash",
		CreatedAt:    time.Now(),
	}
}

func TestMember_Validate_HappyPath(t *testing.T) {
	require.NoError(t, validMember().Validate())
}

func TestMember_Validate_MissingID_ReturnsError(t *testing.T) {
	m := validMember()
	m.ID = ""
	assert.ErrorContains(t, m.Validate(), "ID is required")
}

func TestMember_Validate_MissingHouseholdID_ReturnsError(t *testing.T) {
	m := validMember()
	m.HouseholdID = ""
	assert.ErrorContains(t, m.Validate(), "HouseholdID is required")
}

func TestMember_Validate_MissingDisplayName_ReturnsError(t *testing.T) {
	m := validMember()
	m.DisplayName = ""
	assert.ErrorContains(t, m.Validate(), "DisplayName is required")
}

func TestMember_Validate_MissingEmail_ReturnsError(t *testing.T) {
	m := validMember()
	m.Email = ""
	assert.ErrorContains(t, m.Validate(), "Email is required")
}

func TestMember_Validate_EmailWithoutAt_ReturnsError(t *testing.T) {
	m := validMember()
	m.Email = "notanemail"
	assert.ErrorContains(t, m.Validate(), "Email must contain @")
}

func TestMember_Validate_InvalidRole_ReturnsError(t *testing.T) {
	m := validMember()
	m.Role = "superadmin"
	assert.ErrorContains(t, m.Validate(), "Role")
}

func TestMember_Validate_MultipleErrors_ReturnsAll(t *testing.T) {
	m := member.Member{}
	err := m.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "ID is required")
	assert.ErrorContains(t, err, "HouseholdID is required")
	assert.ErrorContains(t, err, "DisplayName is required")
}

func TestRole_Valid_AllValidRoles(t *testing.T) {
	cases := []struct {
		role  member.Role
		valid bool
	}{
		{member.RoleOwner, true},
		{member.RoleMember, true},
		{member.RoleViewer, true},
		{"admin", false},
		{"", false},
		{"OWNER", false},
	}
	for _, tc := range cases {
		t.Run(string(tc.role), func(t *testing.T) {
			assert.Equal(t, tc.valid, tc.role.Valid())
		})
	}
}
