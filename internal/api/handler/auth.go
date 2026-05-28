package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hearth-ledger/hearth/internal/core/account"
)

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HouseholdID string `json:"household_id"`
		Email       string `json:"email"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.HouseholdID == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "household_id, email, and password are required")
		return
	}

	pair, err := s.auth.Login(r.Context(), account.HouseholdID(req.HouseholdID), req.Email, req.Password)
	if err != nil {
		jsonError(w, err)
		return
	}

	jsonOK(w, map[string]any{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
	})
}

func (s *Server) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh_token is required")
		return
	}

	pair, err := s.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		jsonError(w, err)
		return
	}

	jsonOK(w, map[string]any{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
	})
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh_token is required")
		return
	}

	if err := s.auth.Logout(r.Context(), req.RefreshToken); err != nil {
		jsonError(w, err)
		return
	}
	jsonNoContent(w)
}
