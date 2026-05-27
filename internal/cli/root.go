package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hearth-ledger/hearth/internal/store/sqlite"
)

const (
	defaultDBDir  = ".local/share/hearth"
	defaultDBName = "ledger.db"
	envDBPath     = "HEARTH_DB"
)

// RootCmd is the top-level cobra command for the hearth CLI.
var RootCmd = &cobra.Command{
	Use:           "hearth",
	Short:         "Hearth — household accounting for humans",
	Long:          `A GAAP-compliant double-entry accounting system for households.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	RootCmd.PersistentFlags().String("db", "", "SQLite database path (overrides $"+envDBPath+")")
	RootCmd.PersistentFlags().String("output", "table", "Output format: table, json, csv, plain")

	viper.SetEnvPrefix("HEARTH")
	viper.AutomaticEnv()

	if err := viper.BindPFlag("db", RootCmd.PersistentFlags().Lookup("db")); err != nil {
		panic(err)
	}
}

// DBPath returns the resolved database path: --db flag → $HEARTH_DB → default.
func DBPath() string {
	if p := viper.GetString("db"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultDBName
	}
	return filepath.Join(home, defaultDBDir, defaultDBName)
}

// OpenStore opens the SQLite store at the configured path.
func OpenStore(ctx context.Context) (*sqlite.Store, error) {
	path := DBPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := sqlite.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	return sqlite.New(db), nil
}

// ExitError prints a user-facing error and exits with the given code.
func ExitError(msg string, code int) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}
