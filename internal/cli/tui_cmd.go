package cli

import (
	"github.com/spf13/cobra"

	"github.com/hearth-ledger/hearth/internal/tui/app"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		s, err := OpenStore(ctx)
		if err != nil {
			return err
		}

		householdFlag, _ := cmd.Flags().GetString("household")
		householdID, err := ActiveHouseholdID(householdFlag)
		if err != nil {
			return err
		}

		return app.Start(s, householdID)
	},
}

func init() {
	tuiCmd.Flags().String("household", "", "Household ID to view (overrides config)")
	RootCmd.AddCommand(tuiCmd)
}
