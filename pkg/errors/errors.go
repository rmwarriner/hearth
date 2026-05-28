package errors

import (
	"fmt"
	"strings"
)

// ErrorCode is a machine-readable identifier for an error category.
type ErrorCode string

const (
	ErrGAAPBalance        ErrorCode = "GAAP_BALANCE"
	ErrGAAPMinPostings    ErrorCode = "GAAP_MIN_POSTINGS"
	ErrGAAPLockedPeriod   ErrorCode = "GAAP_LOCKED_PERIOD"
	ErrAccountNotFound    ErrorCode = "ACCOUNT_NOT_FOUND"
	ErrHouseholdNotFound  ErrorCode = "HOUSEHOLD_NOT_FOUND"
	ErrDatabaseConnection ErrorCode = "DATABASE_CONNECTION"
	ErrInvalidAmount      ErrorCode = "INVALID_AMOUNT"
	ErrCurrencyMismatch   ErrorCode = "CURRENCY_MISMATCH"
	ErrNotImplemented     ErrorCode = "NOT_IMPLEMENTED"
	ErrInternal           ErrorCode = "INTERNAL"

	ErrNotFound       ErrorCode = "NOT_FOUND"
	ErrMemberNotFound ErrorCode = "MEMBER_NOT_FOUND"
	ErrUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrForbidden      ErrorCode = "FORBIDDEN"
	ErrTokenExpired   ErrorCode = "TOKEN_EXPIRED"
	ErrTokenInvalid   ErrorCode = "TOKEN_INVALID"
	ErrTokenRevoked   ErrorCode = "TOKEN_REVOKED"
	ErrConflict       ErrorCode = "CONFLICT"
	ErrInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrRateLimited    ErrorCode = "RATE_LIMITED"
)

// HearthError is the user-facing error type. It carries enough context for
// the user to understand what went wrong and how to fix it.
//
// Internal errors between packages use standard fmt.Errorf wrapping.
// Only errors surfaced to the CLI, API, or TUI use HearthError.
type HearthError struct {
	Code      ErrorCode
	Message   string   // what happened — one sentence
	Context   string   // why it happened — one sentence of context
	Hints     []string // numbered recovery options
	HelpTopic string   // hearth help <topic>
	cause     error    // underlying error, not shown to users
}

// New creates a new HearthError.
func New(code ErrorCode, message string) *HearthError {
	return &HearthError{Code: code, Message: message}
}

// WithContext adds the "why it happened" sentence.
func (e *HearthError) WithContext(ctx string) *HearthError {
	e.Context = ctx
	return e
}

// WithHints adds the numbered recovery options.
func (e *HearthError) WithHints(hints ...string) *HearthError {
	e.Hints = hints
	return e
}

// WithHelp sets the help topic string.
func (e *HearthError) WithHelp(topic string) *HearthError {
	e.HelpTopic = topic
	return e
}

// WithCause wraps an underlying error (not shown to users).
func (e *HearthError) WithCause(cause error) *HearthError {
	e.cause = cause
	return e
}

// Error implements the error interface (machine-readable).
func (e *HearthError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/errors.As chaining.
func (e *HearthError) Unwrap() error {
	return e.cause
}

// UserFacing returns the full human-readable error message formatted for
// terminal output, following the Hearth error design spec.
func (e *HearthError) UserFacing() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Error: %s\n", e.Message)

	if e.Context != "" {
		fmt.Fprintf(&b, "  %s\n", e.Context)
	}

	if len(e.Hints) > 0 {
		b.WriteString("\n  To fix this, you can:\n")
		for i, hint := range e.Hints {
			fmt.Fprintf(&b, "    %d. %s\n", i+1, hint)
		}
	}

	if e.HelpTopic != "" {
		fmt.Fprintf(&b, "\n  Learn more: hearth help %s\n", e.HelpTopic)
	}

	return b.String()
}
