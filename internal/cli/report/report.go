package report

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hearth-ledger/hearth/internal/cli"
	"github.com/hearth-ledger/hearth/internal/store"
)

// Cmd is the `hearth report` parent command.
var Cmd = &cobra.Command{
	Use:   "report",
	Short: "Generate financial reports",
}

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Balance sheet as of today (or --as-of date)",
	RunE:  runBalance,
}

func init() {
	balanceCmd.Flags().String("as-of", "", "Report date (YYYY-MM-DD, defaults to today)")
	Cmd.AddCommand(balanceCmd)
}

func runBalance(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	cli.LoadConfig()
	hhOverride, _ := cmd.Root().PersistentFlags().GetString("household")
	hhID, err := cli.ActiveHouseholdID(hhOverride)
	if err != nil {
		return err
	}

	asOfStr, _ := cmd.Flags().GetString("as-of")
	var asOf time.Time
	if asOfStr == "" {
		asOf = time.Now().UTC()
	} else {
		asOf, err = time.Parse(time.DateOnly, asOfStr)
		if err != nil {
			return fmt.Errorf("invalid --as-of date: %w", err)
		}
		asOf = asOf.Add(24*time.Hour - time.Nanosecond)
	}

	s, err := cli.OpenStore(ctx)
	if err != nil {
		return err
	}

	accounts, err := s.ListAccounts(ctx, hhID)
	if err != nil {
		return err
	}

	format, _ := cmd.Root().PersistentFlags().GetString("output")
	out, _ := cli.ParseOutputFormat(format)

	type row struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Currency string `json:"currency"`
		Balance  string `json:"balance"`
	}

	rows := make([]row, 0, len(accounts))
	for _, a := range accounts {
		bal, err := s.GetAccountBalance(ctx, a.ID, asOf)
		if err != nil {
			return fmt.Errorf("get balance for account %s: %w", a.ID, err)
		}
		if !bal.Value.IsZero() || out == cli.FormatTable {
			rows = append(rows, row{
				ID:       string(a.ID),
				Name:     a.Name,
				Type:     string(a.Type),
				Currency: string(a.Currency),
				Balance:  bal.Value.StringFixed(2),
			})
		}
	}

	switch out {
	case cli.FormatJSON:
		return cli.WriteJSON(cmd.OutOrStdout(), map[string]any{
			"as_of":    asOf.Format(time.DateOnly),
			"accounts": rows,
		})
	case cli.FormatCSV:
		cli.WriteCSVRow(cmd.OutOrStdout(), []string{"id", "name", "type", "currency", "balance"})
		for _, r := range rows {
			cli.WriteCSVRow(cmd.OutOrStdout(), []string{r.ID, r.Name, r.Type, r.Currency, r.Balance})
		}
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Balance Sheet as of %s\n\n", asOf.Format(time.DateOnly))
		w := cli.NewTableWriter(cmd.OutOrStdout())
		fmt.Fprintln(w, "ACCOUNT\tTYPE\tCURRENCY\tBALANCE")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Name, r.Type, r.Currency, r.Balance)
		}
		w.Flush()
	}
	return nil
}

// noopQuery satisfies the store.JournalQuery type reference so the import isn't unused.
var _ = store.JournalQuery{}
