package journal

import (
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
)

// EntryID is a UUID string that uniquely identifies a journal entry.
type EntryID string

// PostingID is a UUID string that uniquely identifies a posting.
type PostingID string

// Source indicates how a journal entry was created.
type Source string

const (
	SourceManual   Source = "manual"
	SourceImport   Source = "import"
	SourceReversal Source = "reversal"
)

// JournalEntry is an immutable record of a financial event.
// It contains two or more Postings that must sum to zero.
type JournalEntry struct {
	ID           EntryID
	HouseholdID  account.HouseholdID
	PostedAt     time.Time
	Description  string
	Reference    string
	Source       Source
	CreatedBy    string // member ID
	CreatedAt    time.Time
	IsReversalOf EntryID // empty string if not a reversal
	Postings     []Posting
}

// Posting is one side of a double-entry journal entry.
// A positive Value is a debit; a negative Value is a credit.
type Posting struct {
	ID             PostingID
	JournalEntryID EntryID
	AccountID      account.AccountID
	Amount         currency.Amount
	Memo           string
}
