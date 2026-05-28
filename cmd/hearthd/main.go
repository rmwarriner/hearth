package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/hearth-ledger/hearth/internal/observability"
	"github.com/hearth-ledger/hearth/internal/server"
)

var version = "dev"

func main() {
	var cfgFile string
	flag.StringVar(&cfgFile, "config", "", "path to config file (YAML)")
	flag.Parse()

	cfg, err := server.LoadConfig(cfgFile)
	if err != nil {
		// Logger not yet initialised; write directly to stderr.
		log.Fatal().Err(err).Msg("hearthd: invalid configuration")
	}

	logger := observability.NewLogger(observability.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})
	ctx := logger.WithContext(context.Background())

	log.Ctx(ctx).Info().
		Str("version", version).
		Str("operation", "startup").
		Msg("hearthd starting")

	srv, err := server.New(ctx, cfg)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Str("operation", "startup").Msg("hearthd: failed to initialise server")
	}

	// Serve in a goroutine; block on signal.
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Start(ctx)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Ctx(ctx).Info().
			Str("signal", sig.String()).
			Str("operation", "shutdown").
			Msg("hearthd: signal received")
	case err := <-serveErr:
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("operation", "serve").Msg("hearthd: serve error")
			os.Exit(2)
		}
	}

	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		log.Ctx(ctx).Error().Err(err).Str("operation", "shutdown").Msg("hearthd: shutdown error")
		os.Exit(2)
	}

	log.Ctx(ctx).Info().Str("operation", "shutdown").Msg("hearthd stopped")
}
