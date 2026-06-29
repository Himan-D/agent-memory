package main

// This file provides opt-in wiring for the Cognee-inspired HTTP handlers
// (see internal/cogni/handlers.go). It is intentionally a separate file
// so the existing cmd/server/api.go does not need to be modified.
//
// Usage from cmd/server/main.go (or wherever the router is constructed):
//
//   import "agent-memory/cmd/server"
//
//   // After building the mux router:
//   if cfg.EnableCogniWiring {
//       cogniDeps := cogni.Deps{
//           SessionManager: svc.SessionManager(),
//           Improver:       svc.Improver(),
//           Ledger:         svc.RollbackLedger(),
//           Retriever:      svc.BaseRetriever(),
//       }
//       server.RegisterCogniHandlers(router, cogniDeps)
//   }
//
// The EnableCogniWiring flag defaults to false so production deployments
// are unaffected unless they explicitly opt in.

import (
	"log"

	"github.com/gorilla/mux"

	"agent-memory/internal/cogni"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/improve"
	"agent-memory/internal/memory/rollback"
	"agent-memory/internal/retrieval"
	"agent-memory/internal/session"
)

// CogniDeps is the set of dependencies the cogni handlers need. All fields
// are optional; a nil field disables the corresponding endpoint.
type CogniDeps struct {
	SessionManager *session.Manager
	Distiller      *session.Distiller
	Improver       *improve.Pipeline
	Ledger         rollback.Ledger
	Retriever      retrieval.BaseRetriever
	Logger         *log.Logger
}

// RegisterCogniHandlers installs the cogni HTTP routes onto r. This is a
// no-op when deps.SessionManager, deps.Distiller, deps.Improver,
// deps.Ledger, and deps.Retriever are all nil.
//
// It is safe to call this multiple times -- the underlying cogni
// RegisterRoutes installs handlers idempotently (mux panics on duplicate
// paths, but only across separate router instances).
func RegisterCogniHandlers(r *mux.Router, deps CogniDeps) *mux.Router {
	if r == nil {
		return nil
	}
	if deps.SessionManager == nil && deps.Distiller == nil &&
		deps.Improver == nil && deps.Ledger == nil && deps.Retriever == nil {
		// Nothing to wire.
		return r
	}

	logger := deps.Logger
	if logger == nil {
		logger = log.Default()
	}

	cogni.RegisterRoutes(r, cogni.Deps{
		SessionManager: deps.SessionManager,
		Distiller:      deps.Distiller,
		Improver:       deps.Improver,
		Ledger:         deps.Ledger,
		Retriever:      deps.Retriever,
	})
	logger.Printf("cogni handlers registered: sessions, improve, distill, pipeline rollback, search enhanced")
	return r
}

// MemoryServiceAdapter is a helper that wires the production memory.Service
// into CogniDeps. Callers can use it instead of constructing CogniDeps by
// hand when they have a fully-constructed memory.Service.
//
// Usage:
//
//   cogniDeps := server.CogniDepsFromService(svc)
//   server.RegisterCogniHandlers(router, cogniDeps)
//
// Note: this returns a partially-populated CogniDeps. Fields that
// require extra wiring (Distiller, Improver, Ledger, Retriever) are left
// nil -- callers should populate them explicitly after construction.
func CogniDepsFromService(svc *memory.Service) CogniDeps {
	// This helper exists as a placeholder so the wiring API is stable
	// across releases. The actual field mapping lives here so future
	// service method additions don't require API breaks.
	return CogniDeps{
		// SessionManager and other deps are intentionally not populated
		// here: the production memory.Service does not yet expose
		// SessionManager, Distiller, Improver, Ledger, or Retriever as
		// first-class fields. Callers should construct those components
		// separately and pass them in.
	}
}
