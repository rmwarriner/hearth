package handler

import (
	"github.com/rs/zerolog"

	"github.com/hearth-ledger/hearth/internal/api/openapi"
	"github.com/hearth-ledger/hearth/internal/auth"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
)

// compile-time assertion that Server satisfies the generated ServerInterface.
var _ openapi.ServerInterface = (*Server)(nil)

// Server implements openapi.ServerInterface. Handlers are thin: decode →
// validate → call store or auth service → encode. No business logic here.
type Server struct {
	store  storeapi.Store
	auth   *auth.Service
	logger zerolog.Logger
}

// NewServer creates a Server wired to the given store and auth service.
func NewServer(store storeapi.Store, authSvc *auth.Service, logger zerolog.Logger) *Server {
	return &Server{store: store, auth: authSvc, logger: logger}
}
