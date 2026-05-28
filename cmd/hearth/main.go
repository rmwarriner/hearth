package main

import (
	"fmt"
	"os"

	"github.com/hearth-ledger/hearth/internal/cli"
	"github.com/hearth-ledger/hearth/internal/cli/accounts"
	"github.com/hearth-ledger/hearth/internal/cli/report"
	"github.com/hearth-ledger/hearth/internal/cli/transactions"
)

func main() {
	cli.RootCmd.PersistentFlags().String("household", "", "Override active household ID")

	cli.RootCmd.AddCommand(
		accounts.Cmd,
		transactions.Cmd,
		report.Cmd,
	)

	if err := cli.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
