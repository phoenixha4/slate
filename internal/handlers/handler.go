// Package handlers contains the HTTP request handlers for the REST API.
// Each handler method belongs to Handler, which holds the data-access store
// and a structured logger.  Accepting a store.Store interface (rather than
// a concrete type) makes every handler unit-testable with an in-memory mock.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/phoenixha4/learning_go/internal/store"
)

// Handler wraps application dependencies and exposes them to every route.
type Handler struct {
	store store.Store
	log   *slog.Logger
}

// NewHandler constructs a Handler. log may be nil (a no-op logger is used).
func NewHandler(st store.Store, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{store: st, log: log}
}

// ─── Response helpers ─────────────────────────────────────────────────────

// writeJSON serialises v to JSON and sends it with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Encoding errors are rare (non-marshalable types); log but do nothing
		// because the status header has already been sent.
		_ = err
	}
}

// writeError sends a JSON error body {"error": msg} with the given status.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// decodeBody reads and decodes the request body into dst.
// Returns false and writes a 400 response if decoding fails.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}
