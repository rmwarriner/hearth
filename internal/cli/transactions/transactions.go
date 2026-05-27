package transactions

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/hearth-ledger/hearth/internal/cli"
	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/gaap"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	"github.com/hearth-ledger/hearth/internal/store"
)

// Cmd is the `hearth transactions` parent command.
var Cmd = &cobra.Command{
	Use:   "transactions",
	Short: "Manage journal entries",
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Record a new double-entry transaction",
	Long: `Record a new journal entry with two or more postings.

Each --posting flag takes the form: account-id:amount:currency
For example: --posting acc-checking:-50.00:USD --posting acc-groceries:50.00:USD

Amounts are decimal values; a negative amount is a credit, positive is a debit.`,
	RunE: runAdd,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List journal entries",
	RunE:  runList,
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a journal entry with all its postings",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	addCmd.Flags().String("description", "", "Transaction description (required)")
	addCmd.Flags().String("date", "", "Posted date (YYYY-MM-DD, defaults to today)")
	addCmd.Flags().StringArray("posting", nil, "Posting: account-id:amount:currency (required, at least 2)")
	addCmd.Flags().String("reference", "", "External reference number (optional)")
	_ = addCmd.MarkFlagRequired("description")
	_ = addCmd.MarkFlagRequired("posting")

	listCmd.Flags().String("account", "", "Filter by account ID")
	listCmd.Flags().String("since", "", "Filter entries on or after this date (YYYY-MM-DD)")
	listCmd.Flags().String("until", "", "Filter entries on or before this date (YYYY-MM-DD)")
	listCmd.Flags().String("search", "", "Filter by description substring")
	listCmd.Flags().Int("limit", 50, "Maximum number of entries to return")

	Cmd.AddCommand(addCmd, listCmd, showCmd)
}

func runAdd(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	cli.LoadConfig()
	hhOverride, _ := cmd.Root().PersistentFlags().GetString("household")
	hhID, err := cli.ActiveHouseholdID(hhOverride)
	if err != nil {
		return err
	}

	desc, _ := cmd.Flags().GetString("description")
	dateStr, _ := cmd.Flags().GetString("date")
	ref, _ := cmd.Flags().GetString("reference")
	postingStrs, _ := cmd.Flags().GetStringArray("posting")

	var postedAt time.Time
	if dateStr == "" {
		postedAt = time.Now().UTC().Truncate(24 * time.Hour)
	} else {
		postedAt, err = time.Parse(time.DateOnly, dateStr)
		if err != nil {
			return fmt.Errorf("invalid date %q: use YYYY-MM-DD format", dateStr)
		}
	}

	postings, err := parsePostings(postingStrs)
	if err != nil {
		return err
	}

	entryID := journal.EntryID(uuid.NewString())
	for i := range postings {
		postings[i].ID = journal.PostingID(uuid.NewString())
		postings[i].JournalEntryID = entryID
	}

	entry := journal.JournalEntry{
		ID:          entryID,
		HouseholdID: hhID,
		PostedAt:    postedAt,
		Description: desc,
		Reference:   ref,
		Source:      journal.SourceManual,
		CreatedAt:   time.Now().UTC(),
		Postings:    postings,
	}

	// Build a minimal validation context: accept all referenced accounts as valid
	// (the store's FK constraint enforces actual account existence).
	vctx := gaap.ValidationContext{
		KnownAccounts: make(map[account.AccountID]account.HouseholdID),
	}
	for _, p := range postings {
		vctx.KnownAccounts[p.AccountID] = hhID
	}

	violations := gaap.Validate(entry, vctx)
	if len(violations) > 0 {
		fmt.Fprintln(os.Stderr, "Error: transaction violates GAAP rules")
		for _, v := range violations {
			fmt.Fprintf(os.Stderr, "  - %s\n", v.Error())
			if v.Hint != "" {
				fmt.Fprintf(os.Stderr, "    Hint: %s\n", v.Hint)
			}
		}
		fmt.Fprintln(os.Stderr, "\nLearn more: hearth help gaap")
		os.Exit(3)
	}

	s, err := cli.OpenStore(ctx)
	if err != nil {
		return err
	}

	if err := s.CreateJournalEntry(ctx, entry); err != nil {
		return fmt.Errorf("save transaction: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Transaction recorded: %s (ID: %s)\n", entry.Description, entry.ID)
	return nil
}

func runList(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	cli.LoadConfig()
	hhOverride, _ := cmd.Root().PersistentFlags().GetString("household")
	hhID, err := cli.ActiveHouseholdID(hhOverride)
	if err != nil {
		return err
	}

	accID, _ := cmd.Flags().GetString("account")
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")
	search, _ := cmd.Flags().GetString("search")
	limit, _ := cmd.Flags().GetInt("limit")

	q := store.JournalQuery{
		HouseholdID:     hhID,
		AccountID:       account.AccountID(accID),
		DescriptionLike: search,
		Limit:           limit,
	}
	if sinceStr != "" {
		parsed, parseErr := time.Parse(time.DateOnly, sinceStr)
		if parseErr != nil {
			return fmt.Errorf("invalid --since date: %w", parseErr)
		}
		q.After = parsed
	}
	if untilStr != "" {
		parsed, parseErr := time.Parse(time.DateOnly, untilStr)
		if parseErr != nil {
			return fmt.Errorf("invalid --until date: %w", parseErr)
		}
		q.Before = parsed.Add(24*time.Hour - time.Nanosecond)
	}

	s, err := cli.OpenStore(ctx)
	if err != nil {
		return err
	}

	entries, err := s.ListJournalEntries(ctx, q)
	if err != nil {
		return err
	}

	format, _ := cmd.Root().PersistentFlags().GetString("output")
	out, _ := cli.ParseOutputFormat(format)

	switch out {
	case cli.FormatJSON:
		return cli.WriteJSON(cmd.OutOrStdout(), entries)
	case cli.FormatCSV:
		cli.WriteCSVRow(cmd.OutOrStdout(), []string{"id", "date", "description", "postings"})
		for _, e := range entries {
			cli.WriteCSVRow(cmd.OutOrStdout(), []string{
				string(e.ID),
				e.PostedAt.Format(time.DateOnly),
				e.Description,
				fmt.Sprintf("%d", len(e.Postings)),
			})
		}
	default:
		w := cli.NewTableWriter(cmd.OutOrStdout())
		fmt.Fprintln(w, "ID\tDATE\tDESCRIPTION\tPOSTINGS")
		for _, e := range entries {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
				e.ID, e.PostedAt.Format(time.DateOnly), e.Description, len(e.Postings))
		}
		w.Flush()
	}
	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s, err := cli.OpenStore(ctx)
	if err != nil {
		return err
	}

	entry, err := s.GetJournalEntry(ctx, journal.EntryID(args[0]))
	if err != nil {
		return err
	}

	format, _ := cmd.Root().PersistentFlags().GetString("output")
	out, _ := cli.ParseOutputFormat(format)

	if out == cli.FormatJSON {
		return cli.WriteJSON(cmd.OutOrStdout(), entry)
	}

	w := cli.NewTableWriter(cmd.OutOrStdout())
	fmt.Fprintf(w, "ID:\t%s\n", entry.ID)
	fmt.Fprintf(w, "Date:\t%s\n", entry.PostedAt.Format(time.DateOnly))
	fmt.Fprintf(w, "Description:\t%s\n", entry.Description)
	if entry.Reference != "" {
		fmt.Fprintf(w, "Reference:\t%s\n", entry.Reference)
	}
	fmt.Fprintf(w, "Source:\t%s\n", entry.Source)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "ACCOUNT\tAMOUNT\tCURRENCY\tMEMO")
	for _, p := range entry.Postings {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			p.AccountID, p.Amount.Value.StringFixed(2), p.Amount.Currency, p.Memo)
	}
	w.Flush()
	return nil
}

// parsePostings parses posting strings of the form "account-id:amount:currency".
func parsePostings(strs []string) ([]journal.Posting, error) {
	postings := make([]journal.Posting, 0, len(strs))
	for _, s := range strs {
		parts := strings.SplitN(s, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid posting %q: expected account-id:amount:currency", s)
		}
		accID := strings.TrimSpace(parts[0])
		amountStr := strings.TrimSpace(parts[1])
		cur := strings.TrimSpace(parts[2])

		if accID == "" || amountStr == "" || cur == "" {
			return nil, fmt.Errorf("invalid posting %q: all fields required", s)
		}

		val, err := decimal.NewFromString(amountStr)
		if err != nil {
			return nil, fmt.Errorf("invalid amount %q in posting %q: %w", amountStr, s, err)
		}

		postings = append(postings, journal.Posting{
			AccountID: account.AccountID(accID),
			Amount:    currency.Amount{Value: val, Currency: currency.Currency(cur)},
		})
	}
	return postings, nil
}
