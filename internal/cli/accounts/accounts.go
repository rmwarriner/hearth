package accounts

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hearth-ledger/hearth/internal/cli"
	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
)

// Cmd is the `hearth accounts` parent command.
var Cmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage accounts",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	RunE:  runList,
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new account",
	RunE:  runAdd,
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show account detail and current balance",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	addCmd.Flags().String("name", "", "Account name (required)")
	addCmd.Flags().String("type", "", "Account type: asset, liability, equity, income, expense (required)")
	addCmd.Flags().String("currency", "USD", "Currency (ISO 4217)")
	addCmd.Flags().String("subtype", "", "Account subtype (optional)")
	addCmd.Flags().String("parent", "", "Parent account ID (optional)")
	addCmd.Flags().Bool("placeholder", false, "Mark as placeholder (no direct postings)")
	_ = addCmd.MarkFlagRequired("name")
	_ = addCmd.MarkFlagRequired("type")

	Cmd.AddCommand(listCmd, addCmd, showCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	cli.LoadConfig()
	hhOverride, _ := cmd.Root().PersistentFlags().GetString("household")
	hhID, err := cli.ActiveHouseholdID(hhOverride)
	if err != nil {
		return err
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

	switch out {
	case cli.FormatJSON:
		return cli.WriteJSON(cmd.OutOrStdout(), accounts)
	case cli.FormatCSV:
		cli.WriteCSVRow(cmd.OutOrStdout(), []string{"id", "name", "type", "currency", "subtype"})
		for _, a := range accounts {
			cli.WriteCSVRow(cmd.OutOrStdout(), []string{string(a.ID), a.Name, string(a.Type), string(a.Currency), a.Subtype})
		}
	default:
		w := cli.NewTableWriter(cmd.OutOrStdout())
		fmt.Fprintln(w, "ID\tNAME\tTYPE\tCURRENCY")
		for _, a := range accounts {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, a.Name, a.Type, a.Currency)
		}
		w.Flush()
	}
	return nil
}

func runAdd(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	cli.LoadConfig()
	hhOverride, _ := cmd.Root().PersistentFlags().GetString("household")
	hhID, err := cli.ActiveHouseholdID(hhOverride)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	typ, _ := cmd.Flags().GetString("type")
	cur, _ := cmd.Flags().GetString("currency")
	subtype, _ := cmd.Flags().GetString("subtype")
	parent, _ := cmd.Flags().GetString("parent")
	placeholder, _ := cmd.Flags().GetBool("placeholder")

	at := account.AccountType(typ)
	if !at.Valid() {
		return fmt.Errorf("invalid account type %q: must be asset, liability, equity, income, or expense", typ)
	}

	s, err := cli.OpenStore(ctx)
	if err != nil {
		return err
	}

	a := account.Account{
		ID:            account.AccountID(uuid.NewString()),
		HouseholdID:   hhID,
		Name:          name,
		Type:          at,
		Subtype:       subtype,
		Currency:      currency.Currency(cur),
		ParentID:      account.AccountID(parent),
		IsPlaceholder: placeholder,
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.CreateAccount(ctx, a); err != nil {
		return fmt.Errorf("create account: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Account created: %s (ID: %s)\n", a.Name, a.ID)
	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	s, err := cli.OpenStore(ctx)
	if err != nil {
		return err
	}

	id := account.AccountID(args[0])
	a, err := s.GetAccount(ctx, id)
	if err != nil {
		return err
	}

	bal, err := s.GetAccountBalance(ctx, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	format, _ := cmd.Root().PersistentFlags().GetString("output")
	out, _ := cli.ParseOutputFormat(format)

	if out == cli.FormatJSON {
		return cli.WriteJSON(cmd.OutOrStdout(), map[string]any{
			"id": a.ID, "name": a.Name, "type": a.Type,
			"currency": a.Currency, "balance": bal.Value.String(),
		})
	}

	w := cli.NewTableWriter(cmd.OutOrStdout())
	fmt.Fprintf(w, "ID:\t%s\n", a.ID)
	fmt.Fprintf(w, "Name:\t%s\n", a.Name)
	fmt.Fprintf(w, "Type:\t%s\n", a.Type)
	fmt.Fprintf(w, "Currency:\t%s\n", a.Currency)
	fmt.Fprintf(w, "Balance:\t%s %s\n", bal.Value.StringFixed(2), bal.Currency)
	w.Flush()
	return nil
}
