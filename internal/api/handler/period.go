package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hearth-ledger/hearth/internal/api/openapi"
	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/period"
)

func (s *Server) ListFiscalPeriods(w http.ResponseWriter, r *http.Request, householdId string) {
	periods, err := s.store.ListLockedPeriods(r.Context(), account.HouseholdID(householdId))
	if err != nil {
		jsonError(w, err)
		return
	}
	data := make([]map[string]any, 0, len(periods))
	for _, p := range periods {
		data = append(data, periodJSON(p))
	}
	jsonOK(w, map[string]any{
		"data": data,
		"meta": map[string]any{"total": len(data), "limit": len(data), "offset": 0},
	})
}

func (s *Server) CreateFiscalPeriod(w http.ResponseWriter, r *http.Request, householdId string) {
	var req openapi.CreateFiscalPeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	// openapi_types.Date wraps time.Time; .Time() gives the underlying value.
	start := req.StartDate.Time
	end := req.EndDate.Time

	p := period.FiscalPeriod{
		ID:          period.PeriodID(req.Id),
		HouseholdID: account.HouseholdID(householdId),
		Name:        req.Name,
		StartDate:   start,
		EndDate:     end,
	}

	if err := s.store.CreateFiscalPeriod(r.Context(), p); err != nil {
		jsonError(w, err)
		return
	}
	jsonCreated(w, periodJSON(p))
}

func (s *Server) LockFiscalPeriod(w http.ResponseWriter, r *http.Request, householdId string, periodId string) {
	if err := s.store.LockFiscalPeriod(r.Context(), period.PeriodID(periodId)); err != nil {
		jsonError(w, err)
		return
	}
	// Return a synthetic locked period (full retrieval not in store interface).
	jsonOK(w, map[string]any{"id": periodId, "household_id": householdId})
}

func periodJSON(p period.FiscalPeriod) map[string]any {
	m := map[string]any{
		"id":           string(p.ID),
		"household_id": string(p.HouseholdID),
		"name":         p.Name,
		"start_date":   p.StartDate.Format(time.DateOnly),
		"end_date":     p.EndDate.Format(time.DateOnly),
	}
	if p.LockedAt != nil {
		m["locked_at"] = p.LockedAt
	}
	return m
}
