package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hearth-ledger/hearth/internal/api/openapi"
	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/gaap"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

func (s *Server) ListJournalEntries(w http.ResponseWriter, r *http.Request, householdId string, params openapi.ListJournalEntriesParams) {
	q := storeapi.JournalQuery{HouseholdID: account.HouseholdID(householdId)}
	if params.Limit != nil {
		q.Limit = *params.Limit
	}
	if params.Offset != nil {
		q.Offset = *params.Offset
	}
	if params.AccountId != nil {
		q.AccountID = account.AccountID(*params.AccountId)
	}
	if params.After != nil {
		q.After = *params.After
	}
	if params.Before != nil {
		q.Before = *params.Before
	}
	if params.DescriptionLike != nil {
		q.DescriptionLike = *params.DescriptionLike
	}

	entries, err := s.store.ListJournalEntries(r.Context(), q)
	if err != nil {
		jsonError(w, err)
		return
	}
	data := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		data = append(data, entryJSON(e))
	}
	jsonOK(w, map[string]any{
		"data": data,
		"meta": map[string]any{"total": len(data), "limit": q.Limit, "offset": q.Offset},
	})
}

func (s *Server) CreateJournalEntry(w http.ResponseWriter, r *http.Request, householdId string, _ openapi.CreateJournalEntryParams) {
	var req openapi.CreateJournalEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	postings := make([]journal.Posting, 0, len(req.Postings))
	for _, p := range req.Postings {
		dec, err := parseDecimal(p.Amount)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", "invalid amount: "+p.Amount)
			return
		}
		postings = append(postings, journal.Posting{
			ID:        journal.PostingID(p.Id),
			AccountID: account.AccountID(p.AccountId),
			Amount:    currency.Amount{Value: dec, Currency: currency.Currency(p.Currency)},
			Memo:      ptrStr(p.Memo),
		})
	}

	claims := claimsFromRequest(r)
	e := journal.JournalEntry{
		ID:          journal.EntryID(req.Id),
		HouseholdID: account.HouseholdID(householdId),
		PostedAt:    req.PostedAt,
		Description: req.Description,
		Reference:   ptrStr(req.Reference),
		Source:      journal.SourceManual,
		CreatedBy:   claims,
		Postings:    postings,
	}

	// Load validation context for GAAP guard.
	accounts, err := s.store.ListAccounts(r.Context(), account.HouseholdID(householdId))
	if err != nil {
		jsonError(w, err)
		return
	}
	lockedPeriods, err := s.store.ListLockedPeriods(r.Context(), account.HouseholdID(householdId))
	if err != nil {
		jsonError(w, err)
		return
	}

	vc := buildValidationContext(accounts, lockedPeriods)
	if violations := gaap.Validate(e, vc); len(violations) > 0 {
		msgs := make([]string, len(violations))
		for i, v := range violations {
			msgs[i] = v.Error()
		}
		// Return first violation as the primary message; all violations in hints.
		he := hearth.New(hearth.ErrGAAPBalance, msgs[0])
		for _, m := range msgs[1:] {
			he = he.WithHints(m)
		}
		jsonError(w, he)
		return
	}

	if err := s.store.CreateJournalEntry(r.Context(), e); err != nil {
		jsonError(w, err)
		return
	}
	jsonCreated(w, entryJSON(e))
}

func (s *Server) GetJournalEntry(w http.ResponseWriter, r *http.Request, householdId string, entryId string) {
	e, err := s.store.GetJournalEntry(r.Context(), journal.EntryID(entryId))
	if err != nil {
		jsonError(w, err)
		return
	}
	jsonOK(w, entryJSON(e))
}

func (s *Server) ReverseJournalEntry(w http.ResponseWriter, r *http.Request, householdId string, entryId string) {
	orig, err := s.store.GetJournalEntry(r.Context(), journal.EntryID(entryId))
	if err != nil {
		jsonError(w, err)
		return
	}

	// Build a reversing entry by negating all postings.
	revPostings := make([]journal.Posting, 0, len(orig.Postings))
	for _, p := range orig.Postings {
		newID, err := newID()
		if err != nil {
			jsonError(w, err)
			return
		}
		revPostings = append(revPostings, journal.Posting{
			ID:        journal.PostingID(newID),
			AccountID: p.AccountID,
			Amount:    currency.Amount{Value: p.Amount.Value.Neg(), Currency: p.Amount.Currency},
			Memo:      p.Memo,
		})
	}

	revID, err := newID()
	if err != nil {
		jsonError(w, err)
		return
	}
	claims := claimsFromRequest(r)
	rev := journal.JournalEntry{
		ID:           journal.EntryID(revID),
		HouseholdID:  orig.HouseholdID,
		PostedAt:     timeNowUTC(),
		Description:  "Reversal of " + string(orig.ID),
		Source:       journal.SourceManual,
		CreatedBy:    claims,
		IsReversalOf: orig.ID,
		Postings:     revPostings,
	}

	// GAAP validate the reversal.
	accounts, err := s.store.ListAccounts(r.Context(), orig.HouseholdID)
	if err != nil {
		jsonError(w, err)
		return
	}
	lockedPeriods, err := s.store.ListLockedPeriods(r.Context(), orig.HouseholdID)
	if err != nil {
		jsonError(w, err)
		return
	}
	vc := buildValidationContext(accounts, lockedPeriods)
	if violations := gaap.Validate(rev, vc); len(violations) > 0 {
		jsonError(w, hearth.New(hearth.ErrGAAPBalance, violations[0].Error()))
		return
	}

	if err := s.store.CreateJournalEntry(r.Context(), rev); err != nil {
		jsonError(w, err)
		return
	}
	jsonCreated(w, entryJSON(rev))
}

func entryJSON(e journal.JournalEntry) map[string]any {
	postings := make([]map[string]any, 0, len(e.Postings))
	for _, p := range e.Postings {
		pm := map[string]any{
			"id":         string(p.ID),
			"account_id": string(p.AccountID),
			"amount":     p.Amount.Value.String(),
			"currency":   string(p.Amount.Currency),
			"memo":       p.Memo,
		}
		postings = append(postings, pm)
	}
	m := map[string]any{
		"id":           string(e.ID),
		"household_id": string(e.HouseholdID),
		"posted_at":    e.PostedAt,
		"description":  e.Description,
		"reference":    e.Reference,
		"source":       string(e.Source),
		"created_by":   e.CreatedBy,
		"postings":     postings,
	}
	if e.IsReversalOf != "" {
		m["is_reversal_of"] = string(e.IsReversalOf)
	}
	return m
}
