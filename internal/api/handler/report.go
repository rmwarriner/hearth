package handler

import (
	"net/http"
	"time"

	"github.com/hearth-ledger/hearth/internal/api/openapi"
	"github.com/hearth-ledger/hearth/internal/core/account"
)

// GetBalanceSheet returns the balance for every account as of the given date.
// Balances are computed by summing postings in Go (per ADR-004).
// TODO(phase-7): add materialized balance cache for report performance.
func (s *Server) GetBalanceSheet(w http.ResponseWriter, r *http.Request, householdId string, params openapi.GetBalanceSheetParams) {
	asOf := time.Now().UTC()
	if params.AsOf != nil {
		asOf = *params.AsOf
	}

	accounts, err := s.store.ListAccounts(r.Context(), account.HouseholdID(householdId))
	if err != nil {
		jsonError(w, err)
		return
	}

	lines := make([]map[string]any, 0, len(accounts))
	for _, a := range accounts {
		bal, err := s.store.GetAccountBalance(r.Context(), a.ID, asOf)
		if err != nil {
			jsonError(w, err)
			return
		}
		lines = append(lines, map[string]any{
			"account_id": string(a.ID),
			"name":       a.Name,
			"type":       string(a.Type),
			"currency":   string(bal.Currency),
			"balance":    bal.Value.String(),
		})
	}

	jsonOK(w, map[string]any{"as_of": asOf, "lines": lines})
}

// GetIncomeStatement returns balances for income/expense accounts in a date range.
// TODO(phase-7): add materialized balance cache for report performance.
func (s *Server) GetIncomeStatement(w http.ResponseWriter, r *http.Request, householdId string, params openapi.GetIncomeStatementParams) {
	after := params.After
	before := params.Before

	accounts, err := s.store.ListAccounts(r.Context(), account.HouseholdID(householdId))
	if err != nil {
		jsonError(w, err)
		return
	}

	lines := make([]map[string]any, 0)
	for _, a := range accounts {
		if a.Type != account.Income && a.Type != account.Expense {
			continue
		}
		// Balance as of end of range minus balance as of start of range.
		balEnd, err := s.store.GetAccountBalance(r.Context(), a.ID, before)
		if err != nil {
			jsonError(w, err)
			return
		}
		balStart, err := s.store.GetAccountBalance(r.Context(), a.ID, after)
		if err != nil {
			jsonError(w, err)
			return
		}
		net := balEnd.Value.Sub(balStart.Value)
		lines = append(lines, map[string]any{
			"account_id": string(a.ID),
			"name":       a.Name,
			"type":       string(a.Type),
			"currency":   string(balEnd.Currency),
			"balance":    net.String(),
		})
	}

	jsonOK(w, map[string]any{"after": after, "before": before, "lines": lines})
}
