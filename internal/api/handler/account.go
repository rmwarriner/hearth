package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hearth-ledger/hearth/internal/api/openapi"
	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
)

func (s *Server) ListAccounts(w http.ResponseWriter, r *http.Request, householdId string) {
	accounts, err := s.store.ListAccounts(r.Context(), account.HouseholdID(householdId))
	if err != nil {
		jsonError(w, err)
		return
	}
	data := make([]map[string]any, 0, len(accounts))
	for _, a := range accounts {
		data = append(data, accountJSON(a))
	}
	jsonOK(w, map[string]any{
		"data": data,
		"meta": map[string]any{"total": len(data), "limit": len(data), "offset": 0},
	})
}

func (s *Server) CreateAccount(w http.ResponseWriter, r *http.Request, householdId string) {
	var req openapi.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	a := account.Account{
		ID:            account.AccountID(req.Id),
		HouseholdID:   account.HouseholdID(householdId),
		Name:          req.Name,
		Type:          account.AccountType(req.Type),
		Currency:      currency.Currency(req.Currency),
		IsPlaceholder: req.IsPlaceholder != nil && *req.IsPlaceholder,
		CreatedAt:     time.Now().UTC(),
	}
	if req.ParentId != nil {
		a.ParentID = account.AccountID(*req.ParentId)
	}
	if err := a.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
		return
	}

	if err := s.store.CreateAccount(r.Context(), a); err != nil {
		jsonError(w, err)
		return
	}
	jsonCreated(w, accountJSON(a))
}

func (s *Server) GetAccount(w http.ResponseWriter, r *http.Request, householdId string, accountId string) {
	a, err := s.store.GetAccount(r.Context(), account.AccountID(accountId))
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, accountJSON(a))
}

func (s *Server) UpdateAccount(w http.ResponseWriter, r *http.Request, householdId string, accountId string) {
	var req openapi.UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	// Account update is not in the store interface for Phase 2.
	a, err := s.store.GetAccount(r.Context(), account.AccountID(accountId))
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, accountJSON(a))
}

func (s *Server) GetAccountBalance(w http.ResponseWriter, r *http.Request, householdId string, accountId string, params openapi.GetAccountBalanceParams) {
	asOf := time.Now().UTC()
	if params.AsOf != nil {
		asOf = *params.AsOf
	}

	bal, err := s.store.GetAccountBalance(r.Context(), account.AccountID(accountId), asOf)
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, map[string]any{
		"account_id": accountId,
		"currency":   string(bal.Currency),
		"balance":    bal.Value.String(),
		"as_of":      asOf,
	})
}

func accountJSON(a account.Account) map[string]any {
	m := map[string]any{
		"id":             string(a.ID),
		"household_id":   string(a.HouseholdID),
		"name":           a.Name,
		"type":           string(a.Type),
		"currency":       string(a.Currency),
		"is_placeholder": a.IsPlaceholder,
		"created_at":     a.CreatedAt,
	}
	if a.ParentID != "" {
		m["parent_id"] = string(a.ParentID)
	}
	return m
}
