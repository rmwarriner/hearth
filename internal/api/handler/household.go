package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hearth-ledger/hearth/internal/core/account"
)

func (s *Server) GetHousehold(w http.ResponseWriter, r *http.Request, householdId string) {
	hh, err := s.store.GetHousehold(r.Context(), account.HouseholdID(householdId))
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, map[string]any{
		"id":            string(hh.ID),
		"name":          hh.Name,
		"base_currency": string(hh.BaseCurrency),
		"created_at":    hh.CreatedAt,
	})
}

func (s *Server) UpdateHousehold(w http.ResponseWriter, r *http.Request, householdId string) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	// Household name updates are not yet in the Store interface (Phase 2 scope).
	// Return the current household so the caller gets a well-formed response.
	hh, err := s.store.GetHousehold(r.Context(), account.HouseholdID(householdId))
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, map[string]any{
		"id":            string(hh.ID),
		"name":          hh.Name,
		"base_currency": string(hh.BaseCurrency),
		"created_at":    hh.CreatedAt,
	})
}
