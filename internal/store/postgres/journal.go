package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// CreateJournalEntry inserts a journal entry and all its postings atomically.
func (s *Store) CreateJournalEntry(ctx context.Context, e journal.JournalEntry) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := SetHouseholdContext(ctx, tx, e.HouseholdID); err != nil {
		return err
	}

	reversalOf := sql.NullString{}
	if e.IsReversalOf != "" {
		reversalOf = sql.NullString{String: string(e.IsReversalOf), Valid: true}
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO journal_entries
		 (id, household_id, posted_at, description, reference, source, created_by, is_reversal_of)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		string(e.ID),
		string(e.HouseholdID),
		e.PostedAt.UTC(),
		e.Description,
		e.Reference,
		string(e.Source),
		e.CreatedBy,
		reversalOf,
	)
	if err != nil {
		return fmt.Errorf("insert journal entry: %w", toHearthError(err))
	}

	for _, p := range e.Postings {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO postings (id, journal_entry_id, account_id, amount, currency, memo)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			string(p.ID),
			string(e.ID),
			string(p.AccountID),
			p.Amount.Value.String(),
			string(p.Amount.Currency),
			p.Memo,
		)
		if err != nil {
			return fmt.Errorf("insert posting %s: %w", p.ID, toHearthError(err))
		}
	}

	return tx.Commit()
}

// GetJournalEntry retrieves an entry and all its postings by ID.
func (s *Store) GetJournalEntry(ctx context.Context, id journal.EntryID) (journal.JournalEntry, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, household_id, posted_at, description, reference, source, created_by, is_reversal_of
		 FROM journal_entries WHERE id = $1`,
		string(id),
	)

	e, err := scanEntry(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return journal.JournalEntry{}, hearth.New(hearth.ErrAccountNotFound,
				fmt.Sprintf("journal entry %q not found", id)).
				WithHelp("transactions")
		}
		return journal.JournalEntry{}, fmt.Errorf("get journal entry: %w", err)
	}

	postings, err := s.loadPostings(ctx, e.ID)
	if err != nil {
		return journal.JournalEntry{}, err
	}
	e.Postings = postings
	return e, nil
}

// ListJournalEntries returns journal entries matching the query.
func (s *Store) ListJournalEntries(ctx context.Context, q storeapi.JournalQuery) ([]journal.JournalEntry, error) {
	query, args := buildJournalQuery(q)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}
	defer rows.Close()

	var entries []journal.JournalEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan entry row: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate journal entries: %w", err)
	}

	for i := range entries {
		postings, err := s.loadPostings(ctx, entries[i].ID)
		if err != nil {
			return nil, err
		}
		entries[i].Postings = postings
	}

	return entries, nil
}

func (s *Store) loadPostings(ctx context.Context, entryID journal.EntryID) ([]journal.Posting, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, account_id, amount, currency, memo FROM postings WHERE journal_entry_id = $1`,
		string(entryID),
	)
	if err != nil {
		return nil, fmt.Errorf("load postings for entry %s: %w", entryID, err)
	}
	defer rows.Close()

	var postings []journal.Posting
	for rows.Next() {
		var p journal.Posting
		var pid, accID, amountStr, cur, memo string
		if err := rows.Scan(&pid, &accID, &amountStr, &cur, &memo); err != nil {
			return nil, fmt.Errorf("scan posting row: %w", err)
		}
		val, err := decimal.NewFromString(amountStr)
		if err != nil {
			return nil, fmt.Errorf("parse posting amount %q: %w", amountStr, err)
		}
		p.ID = journal.PostingID(pid)
		p.JournalEntryID = entryID
		p.AccountID = account.AccountID(accID)
		p.Amount = currency.Amount{Value: val, Currency: currency.Currency(cur)}
		p.Memo = memo
		postings = append(postings, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postings: %w", err)
	}
	return postings, nil
}

type entryScanner interface {
	Scan(dest ...any) error
}

func scanEntry(s entryScanner) (journal.JournalEntry, error) {
	var e journal.JournalEntry
	var id, hhID, desc, ref, src, createdBy string
	var postedAt time.Time
	var reversalOf sql.NullString

	err := s.Scan(&id, &hhID, &postedAt, &desc, &ref, &src, &createdBy, &reversalOf)
	if err != nil {
		return journal.JournalEntry{}, err
	}

	e.ID = journal.EntryID(id)
	e.HouseholdID = account.HouseholdID(hhID)
	e.PostedAt = postedAt.UTC()
	e.Description = desc
	e.Reference = ref
	e.Source = journal.Source(src)
	e.CreatedBy = createdBy
	if reversalOf.Valid {
		e.IsReversalOf = journal.EntryID(reversalOf.String)
	}
	return e, nil
}

// buildJournalQuery constructs a parameterised PostgreSQL query from JournalQuery.
// PostgreSQL uses $N placeholders.
func buildJournalQuery(q storeapi.JournalQuery) (string, []any) {
	var where []string
	var args []any
	n := 1

	placeholder := func() string {
		p := fmt.Sprintf("$%d", n)
		n++
		return p
	}

	if q.HouseholdID != "" {
		where = append(where, fmt.Sprintf("je.household_id = %s", placeholder()))
		args = append(args, string(q.HouseholdID))
	}
	if !q.After.IsZero() {
		where = append(where, fmt.Sprintf("je.posted_at >= %s", placeholder()))
		args = append(args, q.After.UTC())
	}
	if !q.Before.IsZero() {
		where = append(where, fmt.Sprintf("je.posted_at <= %s", placeholder()))
		args = append(args, q.Before.UTC())
	}
	if q.DescriptionLike != "" {
		where = append(where, fmt.Sprintf("je.description ILIKE %s", placeholder()))
		args = append(args, "%"+q.DescriptionLike+"%")
	}
	if q.AccountID != "" {
		where = append(where, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM postings p WHERE p.journal_entry_id = je.id AND p.account_id = %s)",
			placeholder(),
		))
		args = append(args, string(q.AccountID))
	}

	baseQuery := `SELECT je.id, je.household_id, je.posted_at, je.description, je.reference,
		je.source, je.created_by, je.is_reversal_of
		FROM journal_entries je`

	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += " ORDER BY je.posted_at DESC"

	if q.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT %s", placeholder())
		args = append(args, q.Limit)
	}
	if q.Offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET %s", placeholder())
		args = append(args, q.Offset)
	}

	return baseQuery, args
}
