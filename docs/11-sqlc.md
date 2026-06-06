# sqlc — The Recommended SQL Layer for Go

This is a learning note for a possible future improvement. The current project uses hand-written `pgx` code, and `sqlc` would be a good next step to explore type-safe SQL in Go.

## What Problem It Solves

The current `internal/store/postgres.go` writes SQL by hand and scans rows manually:

```go
// current approach — error-prone boilerplate
err := rows.Scan(
    &t.ID, &t.Title, &t.Notes,
    &t.ProjectID, &t.ParentID,
    &t.Priority, &t.DueDate,
    &t.Completed, &t.CompletedAt,
    &t.Position, &t.CreatedAt, &t.UpdatedAt,
)
```

If you add a column to the table but forget to update the `Scan` call, you get a **runtime error** — not a compile-time error. The same is true for typos in column names, wrong argument order, and mismatched types.

[sqlc](https://sqlc.dev) solves this: you write SQL, it generates type-safe Go. The scan boilerplate disappears, and mistakes become compile errors.

---

## How It Works

```
your SQL files  →  sqlc generate  →  type-safe Go code
```

1. You write `.sql` files with annotated queries.
2. `sqlc generate` reads your schema and queries, then emits:
   - A Go struct per table row (`Task`, `Project`, etc.)
   - A typed function per query (`ListTasks`, `CreateTask`, etc.)
   - A `Querier` interface you can mock in tests.

Your handlers and store then call the generated code — no `rows.Scan`, no hand-built `args` slices.

---

## Example

### `sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/store/queries/"
    schema:  "internal/db/schema.sql"
    gen:
      go:
        package:       "store"
        out:           "internal/store/sqlc"
        emit_interface: true
```

### `internal/store/queries/tasks.sql`

```sql
-- name: GetTask :one
SELECT id, title, notes, project_id, parent_id, priority,
       to_char(due_date, 'YYYY-MM-DD') AS due_date,
       completed, completed_at, position, created_at, updated_at
FROM tasks
WHERE id = $1::uuid;

-- name: ListTasks :many
SELECT id, title, notes, project_id, parent_id, priority,
       to_char(due_date, 'YYYY-MM-DD') AS due_date,
       completed, completed_at, position, created_at, updated_at
FROM tasks
WHERE parent_id IS NULL
ORDER BY position, created_at;

-- name: CreateTask :one
INSERT INTO tasks (title, notes, project_id, parent_id, priority, due_date, position)
VALUES ($1, $2, $3::uuid, $4::uuid, $5, $6::date,
        COALESCE((SELECT MAX(position) FROM tasks WHERE project_id IS NOT DISTINCT FROM $3::uuid), 0) + 1)
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks WHERE id = $1::uuid;
```

### What sqlc generates

```go
// generated — do not edit

type GetTaskRow struct {
    ID          string
    Title       string
    Notes       string
    ProjectID   pgtype.UUID
    ParentID    pgtype.UUID
    Priority    int32
    DueDate     pgtype.Text
    Completed   bool
    CompletedAt pgtype.Timestamptz
    Position    int32
    CreatedAt   pgtype.Timestamptz
    UpdatedAt   pgtype.Timestamptz
}

func (q *Queries) GetTask(ctx context.Context, id pgtype.UUID) (GetTaskRow, error) {
    row := q.db.QueryRow(ctx, getTask, id)
    var i GetTaskRow
    err := row.Scan(
        &i.ID, &i.Title, &i.Notes, &i.ProjectID, &i.ParentID,
        &i.Priority, &i.DueDate, &i.Completed, &i.CompletedAt,
        &i.Position, &i.CreatedAt, &i.UpdatedAt,
    )
    return i, err
}
```

You never write or touch that `Scan` call — sqlc owns it.

---

## When to Use sqlc vs Raw pgx

| Scenario | Use |
|---|---|
| Standard CRUD (insert, select by ID, delete) | **sqlc** — zero boilerplate, compile-time safe |
| Partial UPDATE with optional fields (`PATCH`) | **Raw pgx** — dynamic `SET` clause is hard to express in static SQL |
| Bulk operations, `COPY FROM` | **Raw pgx** — sqlc doesn't generate batch helpers for all patterns |
| Complex reporting / analytics queries | **sqlc** — SQL is the source of truth, generated function is clean |
| Transactions spanning multiple queries | **sqlc** (with `WithTx`) or **raw pgx** — both work fine |

In this repo, `ListTasks` (dynamic filter building) and `UpdateTask`/`UpdateProject` (dynamic `SET`) are the two places where raw pgx remains the right choice. Everything else — `GetTask`, `CreateTask`, `DeleteTask`, `ListProjects`, etc. — is a perfect sqlc candidate.

---

## Migration Path for This Repo

1. `brew install sqlc` (or `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
2. Extract the `schema` constant from `internal/db/db.go` into `internal/db/schema.sql`
3. Write query files under `internal/store/queries/`
4. Add `sqlc.yaml` at the repo root
5. Run `sqlc generate` — it creates `internal/store/sqlc/`
6. Replace the hand-written `postgres.go` methods one-by-one with calls to the generated `Queries` struct
7. Keep the existing `Store` interface — the generated `Querier` can satisfy it (or you wrap it)

---

## Resources

- Docs: https://docs.sqlc.dev
- Playground: https://play.sqlc.dev (paste your schema + SQL, see generated Go instantly)
- pgx v5 support: `sqlc` has first-class support for `pgx/v5` — set `sql_driver: "pgx/v5"` in `sqlc.yaml`
