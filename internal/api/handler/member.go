package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/member"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

func (s *Server) ListMembers(w http.ResponseWriter, r *http.Request, householdId string) {
	members, err := s.store.ListMembers(r.Context(), account.HouseholdID(householdId))
	if err != nil {
		jsonError(w, err)
		return
	}
	data := make([]map[string]any, 0, len(members))
	for _, m := range members {
		data = append(data, memberJSON(m))
	}
	jsonOK(w, map[string]any{
		"data": data,
		"meta": map[string]any{"total": len(data), "limit": len(data), "offset": 0},
	})
}

func (s *Server) CreateMember(w http.ResponseWriter, r *http.Request, householdId string) {
	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Role        string `json:"role"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	hash, err := s.auth.HashNewPassword(req.Password)
	if err != nil {
		jsonError(w, err)
		return
	}

	id, err := newID()
	if err != nil {
		jsonError(w, err)
		return
	}

	m := member.Member{
		ID:           member.MemberID(id),
		HouseholdID:  account.HouseholdID(householdId),
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Role:         member.Role(req.Role),
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	if err := m.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
		return
	}

	if err := s.store.CreateMember(r.Context(), m); err != nil {
		jsonError(w, err)
		return
	}
	jsonCreated(w, memberJSON(m))
}

func (s *Server) GetMember(w http.ResponseWriter, r *http.Request, householdId string, memberId string) {
	m, err := s.store.GetMember(r.Context(), member.MemberID(memberId))
	if err != nil {
		jsonError(w, err)
		return
	}
	if string(m.HouseholdID) != householdId {
		jsonError(w, hearth.New(hearth.ErrMemberNotFound, "member not found in this household"))
		return
	}
	jsonOK(w, memberJSON(m))
}

func (s *Server) UpdateMember(w http.ResponseWriter, r *http.Request, householdId string, memberId string) {
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	role := member.Role(req.Role)
	if !role.Valid() {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", "invalid role")
		return
	}
	if err := s.store.UpdateMemberRole(r.Context(), member.MemberID(memberId), role); err != nil {
		jsonError(w, err)
		return
	}
	m, err := s.store.GetMember(r.Context(), member.MemberID(memberId))
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, memberJSON(m))
}

func (s *Server) DeleteMember(w http.ResponseWriter, r *http.Request, householdId string, memberId string) {
	// Member deletion is not in the store interface for Phase 2.
	// Return 501 with a clear message.
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "member deletion is not yet implemented")
}

func memberJSON(m member.Member) map[string]any {
	return map[string]any{
		"id":           string(m.ID),
		"household_id": string(m.HouseholdID),
		"display_name": m.DisplayName,
		"email":        m.Email,
		"role":         string(m.Role),
		"created_at":   m.CreatedAt,
	}
}
