package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Server wraps an http.Server with graceful shutdown.
type Server struct {
	http    *http.Server
	cleanup func()
}

// New creates a Server from config. Call Start to begin serving.
func New(ctx context.Context, cfg Config) (*Server, error) {
	handler, cleanup, err := Build(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Server{
		http:    &http.Server{Addr: cfg.ListenAddr, Handler: handler},
		cleanup: cleanup,
	}, nil
}

// Start begins listening. It blocks until the context is cancelled or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.http.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.http.Addr, err)
	}

	log.Ctx(ctx).Info().
		Str("addr", ln.Addr().String()).
		Str("operation", "startup").
		Msg("hearthd listening")

	if err := s.http.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// Shutdown drains in-flight requests with a 10-second timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	shutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	defer s.cleanup()

	log.Ctx(ctx).Info().Str("operation", "shutdown").Msg("hearthd shutting down")

	if err := s.http.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}
