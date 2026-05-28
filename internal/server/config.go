package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration for hearthd.
type Config struct {
	ListenAddr       string        `mapstructure:"listen_addr"`
	LogLevel         string        `mapstructure:"log_level"`
	LogFormat        string        `mapstructure:"log_format"`
	DatabaseURL      string        `mapstructure:"database_url"`
	JWTSecret        string        `mapstructure:"jwt_secret"`
	AccessTokenTTL   time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `mapstructure:"refresh_token_ttl"`
	BcryptCost       int           `mapstructure:"bcrypt_cost"`
	DBMaxConns       int32         `mapstructure:"db_max_conns"`
	DBMinConns       int32         `mapstructure:"db_min_conns"`
	DBConnectTimeout time.Duration `mapstructure:"db_connect_timeout"`
}

// LoadConfig reads configuration from a YAML file and environment variables.
// Environment variable prefix: HEARTH_ (e.g. HEARTH_DATABASE_URL).
// Hard aborts if HEARTH_DB_URL or HEARTH_JWT_SECRET is missing or weak.
func LoadConfig(cfgFile string) (Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("listen_addr", ":8080")
	v.SetDefault("log_level", "info")
	v.SetDefault("log_format", "json")
	v.SetDefault("access_token_ttl", 15*time.Minute)
	v.SetDefault("refresh_token_ttl", 7*24*time.Hour)
	v.SetDefault("bcrypt_cost", 12)
	v.SetDefault("db_max_conns", 10)
	v.SetDefault("db_min_conns", 1)
	v.SetDefault("db_connect_timeout", 10*time.Second)

	v.SetEnvPrefix("HEARTH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("read config file: %w", err)
		}
	}

	// HEARTH_DATABASE_URL and HEARTH_JWT_SECRET can come from env even without a file.
	if url := v.GetString("database_url"); url == "" {
		if envURL := v.GetString("db_url"); envURL != "" {
			v.Set("database_url", envURL)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("HEARTH_DATABASE_URL (or database_url in config) is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("HEARTH_JWT_SECRET (or jwt_secret in config) is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("HEARTH_JWT_SECRET must be at least 32 characters")
	}

	return cfg, nil
}
