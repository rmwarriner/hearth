package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/hearth-ledger/hearth/internal/api/handler"
	"github.com/hearth-ledger/hearth/internal/api/middleware"
	"github.com/hearth-ledger/hearth/internal/api/openapi"
	"github.com/hearth-ledger/hearth/internal/auth"
	"github.com/hearth-ledger/hearth/internal/observability"
	pgstore "github.com/hearth-ledger/hearth/internal/store/postgres"
)

// Build wires together all dependencies and returns a ready http.Handler.
func Build(ctx context.Context, cfg Config) (http.Handler, func(), error) {
	logger := observability.NewLogger(observability.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})

	db, err := pgstore.Open(ctx, pgstore.Config{
		DSN:            cfg.DatabaseURL,
		MaxConns:       cfg.DBMaxConns,
		MinConns:       cfg.DBMinConns,
		ConnectTimeout: cfg.DBConnectTimeout,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres: %w", err)
	}
	cleanup := func() { db.Close() }

	store := pgstore.New(db)

	authSvc, err := auth.NewService(store, auth.Config{
		JWTSecret:       []byte(cfg.JWTSecret),
		AccessTokenTTL:  cfg.AccessTokenTTL,
		RefreshTokenTTL: cfg.RefreshTokenTTL,
		BcryptCost:      cfg.BcryptCost,
	}, logger)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("create auth service: %w", err)
	}

	srv := handler.NewServer(store, authSvc, logger)

	r := chi.NewRouter()
	r.Use(middleware.Recovery)
	r.Use(middleware.RequestLogger(logger))
	r.Use(chiMiddleware.RequestID)

	r.Route("/api/v1", func(r chi.Router) {
		// Unauthenticated auth endpoints.
		r.Post("/auth/login", srv.Login)
		r.Post("/auth/refresh", srv.RefreshToken)

		// Authenticated routes.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(authSvc))
			r.Post("/auth/logout", srv.Logout)

			r.Route("/households/{householdId}", func(r chi.Router) {
				r.Use(middleware.VerifyHousehold)

				// Wire all generated routes via chi handler.
				openapi.HandlerFromMux(srv, r)
			})
		})
	})

	return r, cleanup, nil
}

// defaultTimeout is the graceful-shutdown deadline.
const defaultTimeout = 10 * time.Second
