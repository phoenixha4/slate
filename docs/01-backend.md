# Backend (Go)

The backend is intentionally small and learning-focused. It uses the Go standard library for HTTP routing and `pgx/v5` for PostgreSQL access.

## Packages

| Package | Purpose |
|---------|---------|
| `cmd/server` | Entry point — wires config → DB → store → server |
| `internal/config` | Environment-based config (PORT, DATABASE_URL) |
| `internal/db` | PostgreSQL connection pool + auto-migrations |
| `internal/models` | Domain types (Project, Label, Task, inputs) |
| `internal/store` | Store interface + PostgreSQL implementation |
| `internal/handlers` | HTTP handlers (REST endpoints) |
| `internal/middleware` | Logger, CORS, Recovery, Chain |
| `internal/server` | HTTP mux registration + middleware wrapping |
| `assets` | `go:embed` for frontend static files |

## API routes

All routes are relative to the server root (the shared proxy routes `/` to this service).

### Projects
- `GET /projects` — list all non-archived projects
- `POST /projects` — create project
- `GET /projects/{id}` — get single project
- `PATCH /projects/{id}` — partial update
- `DELETE /projects/{id}` — delete (Inbox returns 403)

### Labels
- `GET /labels` — list all labels
- `POST /labels` — create label
- `PATCH /labels/{id}` — partial update
- `DELETE /labels/{id}` — delete

### Tasks
- `GET /tasks` — list tasks (query: `project_id`, `due=today|upcoming`, `completed=true|false`)
- `GET /tasks/search?q=...` — search by title/notes
- `POST /tasks` — create task
- `GET /tasks/{id}` — get task with labels and subtasks
- `PATCH /tasks/{id}` — partial update (title, notes, project, priority, due, completed, labels)
- `DELETE /tasks/{id}` — delete task + subtasks

## Middleware chain

```go
middleware.Chain(mux,
    middleware.Recover(log),   // catch panics and return 500
    middleware.Logger(log),   // structured request logging
    middleware.CORS,          // config-driven CORS
)
```

## Database migrations

Forward-only, idempotent DDL in `internal/db/db.go`. Runs on every startup:
- `CREATE TABLE IF NOT EXISTS` for all tables
- `CREATE INDEX IF NOT EXISTS` for all indexes
- Seed Inbox project with fixed UUID (`00000000-0000-0000-0000-000000000001`)

## Testing

Unit tests in `internal/handlers/tasks_test.go` use an in-memory `mockStore` implementing the `store.Store` interface. No database needed for handler tests.

```bash
go test ./internal/handlers/
```
