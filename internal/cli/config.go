package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/hearth-ledger/hearth/internal/core/account"
)

const (
	configDir  = ".config/hearth"
	configFile = "config.yaml"
	envConfig  = "HEARTH_CONFIG"
)

// ConfigPath returns the path to the Hearth config file.
// $HEARTH_CONFIG overrides the default (~/.config/hearth/config.yaml).
func ConfigPath() string {
	if p := os.Getenv(envConfig); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return configFile
	}
	return filepath.Join(home, configDir, configFile)
}

// ActiveHouseholdID returns the active household ID from config or the --household flag.
func ActiveHouseholdID(override string) (account.HouseholdID, error) {
	if override != "" {
		return account.HouseholdID(override), nil
	}
	id := viper.GetString("household_id")
	if id == "" {
		return "", fmt.Errorf("no active household: run `hearth init` to create one")
	}
	return account.HouseholdID(id), nil
}

// WriteConfig persists key=value to the Hearth config file.
// It only writes keys explicitly set by the application, not env-var-derived keys.
func WriteConfig(key, value string) error {
	cp := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(cp), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Use a fresh viper instance to avoid writing env-var-derived keys.
	v := viper.New()
	v.SetConfigFile(cp)
	v.SetConfigType("yaml")
	_ = v.ReadInConfig() // load existing values so we merge, not overwrite
	v.Set(key, value)
	return v.WriteConfigAs(cp)
}

// LoadConfig loads the Hearth config file if it exists.
func LoadConfig() {
	cp := ConfigPath()
	viper.SetConfigFile(cp)
	viper.SetConfigType("yaml")
	_ = viper.ReadInConfig() // ignore "file not found" errors
}
