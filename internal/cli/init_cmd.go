package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/household"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new household database",
	Long:  `Create a new Hearth database at the configured path and initialize a household.`,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().String("name", "My Household", "Household name")
	initCmd.Flags().String("currency", "USD", "Base currency (ISO 4217)")
	RootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()
	path := DBPath()

	// Refuse to reinitialise an existing database
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Database already exists at %s\n", path)
		fmt.Fprintln(cmd.OutOrStdout(), "Run `hearth accounts list` to verify your existing data.")
		return nil
	}

	s, err := OpenStore(ctx)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	name, _ := cmd.Flags().GetString("name")
	cur, _ := cmd.Flags().GetString("currency")

	hh := household.Household{
		ID:              account.HouseholdID(uuid.NewString()),
		Name:            name,
		FiscalYearStart: 1,
		BaseCurrency:    currency.Currency(cur),
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.CreateHousehold(ctx, hh); err != nil {
		return fmt.Errorf("create household: %w", err)
	}

	if err := WriteConfig("household_id", string(hh.ID)); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Initialized Hearth database at %s\n", path)
	fmt.Fprintf(cmd.OutOrStdout(), "Household: %s (ID: %s)\n", hh.Name, hh.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "Base currency: %s\n", hh.BaseCurrency)
	fmt.Fprintln(cmd.OutOrStdout(), "\nNext steps:")
	fmt.Fprintln(cmd.OutOrStdout(), "  hearth accounts add --name \"Checking\" --type asset")
	fmt.Fprintln(cmd.OutOrStdout(), "  hearth transactions add")
	return nil
}
