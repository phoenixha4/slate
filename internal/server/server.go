// Package server wires together the HTTP mux, middleware, static file serving,
// and all API routes into a single http.Handler ready for http.Server.
package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/phoenixha4/learning_go/assets"
	"github.com/phoenixha4/learning_go/internal/handlers"
	"github.com/phoenixha4/learning_go/internal/middleware"
	"github.com/phoenixha4/learning_go/internal/store"
)

// Options configures process-level HTTP behavior.
type Options struct {
	CORSAllowedOrigins []string
	ReadinessTimeout   time.Duration
}

// New builds and returns the root HTTP handler. It:
//   - Registers all /* routes.
//   - Serves the embedded frontend from the assets package.
//   - Wraps everything with recovery, structured logging, and CORS middleware.
func New(st store.Store, log *slog.Logger, opts Options) http.Handler {
	if opts.ReadinessTimeout == 0 {
		opts.ReadinessTimeout = 2 * time.Second
	}

	mux := http.NewServeMux()
	h := handlers.NewHandler(st, log)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), opts.ReadinessTimeout)
		defer cancel()
		if err := st.Ping(ctx); err != nil {
			log.Error("readiness check failed", "error", err)
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}` + "\n"))
	})

	// ─── Project routes ────────────────────────────────────────────────────
	mux.HandleFunc("GET /projects", h.ListProjects)
	mux.HandleFunc("POST /projects", h.CreateProject)
	mux.HandleFunc("GET /projects/{id}", h.GetProject)
	mux.HandleFunc("PATCH /projects/{id}", h.UpdateProject)
	mux.HandleFunc("DELETE /projects/{id}", h.DeleteProject)

	// ─── Label routes ──────────────────────────────────────────────────────
	mux.HandleFunc("GET /labels", h.ListLabels)
	mux.HandleFunc("POST /labels", h.CreateLabel)
	mux.HandleFunc("PATCH /labels/{id}", h.UpdateLabel)
	mux.HandleFunc("DELETE /labels/{id}", h.DeleteLabel)

	// ─── Task routes ───────────────────────────────────────────────────────
	// Note: /tasks/search must be registered before /tasks/{id} so
	// the literal path wins over the wildcard pattern.
	mux.HandleFunc("GET /tasks/search", h.SearchTasks)
	mux.HandleFunc("GET /tasks", h.ListTasks)
	mux.HandleFunc("POST /tasks", h.CreateTask)
	mux.HandleFunc("GET /tasks/{id}", h.GetTask)
	mux.HandleFunc("PATCH /tasks/{id}", h.UpdateTask)
	mux.HandleFunc("DELETE /tasks/{id}", h.DeleteTask)

	// Strip the "frontend/" prefix so that the embedded frontend maps to "/".
	frontendFS, err := fs.Sub(assets.FS, "frontend")
	if err != nil {
		panic("failed to sub frontend FS: " + err.Error())
	}
	mux.Handle("/", http.FileServer(http.FS(frontendFS)))

	// otelhttp wraps the mux: each request gets a trace span and RED metrics
	// (requests, errors, duration) automatically recorded against the route.
	// middleware.Logger is replaced by otelhttp — request logs flow through
	// the OTEL slog bridge wired in telemetry.Setup.
	instrumented := otelhttp.NewHandler(mux, "server",
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
	)
	return middleware.Chain(instrumented,
		middleware.Recover(log),
		middleware.Logger(log),
		middleware.CORS(opts.CORSAllowedOrigins),
	)
}
