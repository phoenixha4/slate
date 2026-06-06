# Database Schema

PostgreSQL 16. Auto-migrated on every startup via `CREATE TABLE IF NOT EXISTS`.

## Tables

### `projects`
```sql
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
```

### `labels`
```sql
CREATE TABLE IF NOT EXISTS labels (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    color       VARCHAR(7)   NOT NULL DEFAULT '#888899',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

### `tasks`
```sql
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
```

Priority levels: 1 = urgent (red), 2 = high (orange), 3 = medium (blue), 4 = none (gray).

### `task_labels`
```sql
CREATE TABLE IF NOT EXISTS task_labels (
    task_id   UUID NOT NULL REFERENCES tasks(id)  ON DELETE CASCADE,
    label_id  UUID NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, label_id)
);
```

## Indexes

```sql
CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id  ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date   ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_completed  ON tasks(completed);
CREATE INDEX IF NOT EXISTS idx_task_labels_task ON task_labels(task_id);
```

## Seed data

The Inbox project is seeded with a fixed UUID so it survives restarts:
```sql
INSERT INTO projects (id, name, color, icon, position)
VALUES ('00000000-0000-0000-0000-000000000001', 'Inbox', '#7c6af7', 'inbox', 0)
ON CONFLICT (id) DO NOTHING;
```

## Design Notes

- **UUID primary keys** - globally unique, safe for distributed systems, and harder to guess than sequential IDs.
- **Self-referencing `tasks.parent_id`** - enables subtask hierarchy.
- **Integer `position`** - leaves room for manual ordering later.
- **`task_count` in API** - computed on read via `COUNT(t.id) FILTER (WHERE NOT t.completed)` in `ListProjects`.
- **`ON DELETE CASCADE`** - deleting a project deletes its tasks; deleting a label removes its task associations.
- **`pgcrypto` extension** - required for `gen_random_uuid()` on PostgreSQL 13 and earlier.
