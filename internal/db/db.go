// Package db provides the PostgreSQL connection pool and runs schema migrations
// at startup so the application always starts against a ready database.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a pgxpool connection pool, verifies connectivity with a
// Ping, then applies forward-only schema migrations before returning.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return pool, nil
}

// migrate runs CREATE TABLE IF NOT EXISTS for every application table and
// seeds the default Inbox project. It is idempotent and safe to run on
// every startup.
func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, schema)
	return err
}

// schema is the complete DDL for the application. All statements use
// IF NOT EXISTS / ON CONFLICT so they are safe to run repeatedly.
const schema = `
-- pgcrypto supplies gen_random_uuid() on Postgres 13 and earlier;
-- Postgres 14+ has gen_random_uuid() built-in, but the extension is harmless.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- projects ─────────────────────────────────────────────────────────────────
-- A named collection of tasks with a colour and an icon slug.
CREATE TABLE IF NOT EXISTS projects (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    color       VARCHAR(7)   NOT NULL DEFAULT '#7c6af7',
    icon        VARCHAR(50)  NOT NULL DEFAULT 'folder',
    position    INTEGER      NOT NULL DEFAULT 0,
    archived    BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- labels ──────────────────────────────────────────────────────────────────
-- Coloured tags that can be attached to any number of tasks.
CREATE TABLE IF NOT EXISTS labels (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    color       VARCHAR(7)   NOT NULL DEFAULT '#888899',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- tasks ───────────────────────────────────────────────────────────────────
-- Core entity.  parent_id enables an arbitrary subtask hierarchy.
-- priority: 1 = urgent (P1), 2 = high (P2), 3 = medium (P3), 4 = none (P4).
CREATE TABLE IF NOT EXISTS tasks (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    title        VARCHAR(500) NOT NULL,
    notes        TEXT         NOT NULL DEFAULT '',
    project_id   UUID         REFERENCES projects(id) ON DELETE CASCADE,
    parent_id    UUID         REFERENCES tasks(id)    ON DELETE CASCADE,
    priority     INTEGER      NOT NULL DEFAULT 4 CHECK (priority BETWEEN 1 AND 4),
    due_date     DATE,
    completed    BOOLEAN      NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    position     INTEGER      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- task_labels ─────────────────────────────────────────────────────────────
-- Many-to-many join table between tasks and labels.
CREATE TABLE IF NOT EXISTS task_labels (
    task_id   UUID NOT NULL REFERENCES tasks(id)  ON DELETE CASCADE,
    label_id  UUID NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, label_id)
);

-- Indexes for common access patterns
CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id  ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date   ON tasks(due_date)   WHERE due_date IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_completed  ON tasks(completed);
CREATE INDEX IF NOT EXISTS idx_task_labels_task ON task_labels(task_id);

-- Seed: the default Inbox project uses a fixed UUID so it survives restarts.
INSERT INTO projects (id, name, color, icon, position)
VALUES ('00000000-0000-0000-0000-000000000001', 'Inbox', '#7c6af7', 'inbox', 0)
ON CONFLICT (id) DO NOTHING;
`
