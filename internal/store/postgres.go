// Package store — PostgreSQL implementation of the Store interface.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/phoenixha4/slate/internal/models"
)

// inboxProjectID is the fixed UUID for the built-in Inbox project seeded in
// the migration. It cannot be deleted and is the default project for new tasks.
const inboxProjectID = "00000000-0000-0000-0000-000000000001"

// PostgresStore implements Store using a pgxpool connection pool.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore wraps an existing pgxpool.Pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Ping verifies that the backing PostgreSQL connection is reachable.
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// ─── Projects ─────────────────────────────────────────────────────────────

// ListProjects returns all non-archived projects ordered by position,
// each augmented with a count of active (non-completed) top-level tasks.
func (s *PostgresStore) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			p.id::text,
			p.name,
			p.color,
			p.icon,
			p.position,
			p.archived,
			COUNT(t.id) FILTER (WHERE NOT t.completed) AS task_count,
			p.created_at,
			p.updated_at
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id AND t.parent_id IS NULL
		WHERE NOT p.archived
		GROUP BY p.id
		ORDER BY p.position, p.created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]models.Project, 0)
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Color, &p.Icon,
			&p.Position, &p.Archived, &p.TaskCount,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// GetProject fetches a single project by ID. Returns (nil, nil) if not found.
func (s *PostgresStore) GetProject(ctx context.Context, id string) (*models.Project, error) {
	var p models.Project
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, name, color, icon, position, archived, created_at, updated_at
		FROM projects WHERE id = $1::uuid
	`, id).Scan(&p.ID, &p.Name, &p.Color, &p.Icon, &p.Position, &p.Archived, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &p, nil
}

// CreateProject inserts a new project with the next available position.
func (s *PostgresStore) CreateProject(ctx context.Context, in models.CreateProjectInput) (*models.Project, error) {
	color := in.Color
	if color == "" {
		color = "#7c6af7"
	}
	icon := in.Icon
	if icon == "" {
		icon = "folder"
	}

	var p models.Project
	err := s.pool.QueryRow(ctx, `
		INSERT INTO projects (name, color, icon, position)
		VALUES ($1, $2, $3,
			COALESCE((SELECT MAX(position) FROM projects), 0) + 1
		)
		RETURNING id::text, name, color, icon, position, archived, created_at, updated_at
	`, in.Name, color, icon).Scan(
		&p.ID, &p.Name, &p.Color, &p.Icon,
		&p.Position, &p.Archived, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return &p, nil
}

// UpdateProject applies a partial update to a project. Returns (nil, nil) if not found.
func (s *PostgresStore) UpdateProject(ctx context.Context, id string, in models.UpdateProjectInput) (*models.Project, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	n := 1

	if in.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", n))
		args = append(args, *in.Name)
		n++
	}
	if in.Color != nil {
		sets = append(sets, fmt.Sprintf("color = $%d", n))
		args = append(args, *in.Color)
		n++
	}
	if in.Icon != nil {
		sets = append(sets, fmt.Sprintf("icon = $%d", n))
		args = append(args, *in.Icon)
		n++
	}
	if in.Archived != nil {
		sets = append(sets, fmt.Sprintf("archived = $%d", n))
		args = append(args, *in.Archived)
		n++
	}

	args = append(args, id)
	var p models.Project
	err := s.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE projects SET %s WHERE id = $%d::uuid
		RETURNING id::text, name, color, icon, position, archived, created_at, updated_at
	`, strings.Join(sets, ", "), n), args...).Scan(
		&p.ID, &p.Name, &p.Color, &p.Icon,
		&p.Position, &p.Archived, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return &p, nil
}

// DeleteProject removes a project and all its tasks (via ON DELETE CASCADE).
// The built-in Inbox project cannot be deleted.
func (s *PostgresStore) DeleteProject(ctx context.Context, id string) error {
	if id == inboxProjectID {
		return ErrInboxProtected
	}
	tag, err := s.pool.Exec(ctx, "DELETE FROM projects WHERE id = $1::uuid", id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Labels ───────────────────────────────────────────────────────────────

// ListLabels returns all labels ordered alphabetically.
func (s *PostgresStore) ListLabels(ctx context.Context) ([]models.Label, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, color, created_at FROM labels ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	defer rows.Close()

	labels := make([]models.Label, 0)
	for rows.Next() {
		var l models.Label
		if err := rows.Scan(&l.ID, &l.Name, &l.Color, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan label: %w", err)
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

// CreateLabel inserts a new label. Name must be globally unique.
func (s *PostgresStore) CreateLabel(ctx context.Context, in models.CreateLabelInput) (*models.Label, error) {
	color := in.Color
	if color == "" {
		color = "#888899"
	}
	var l models.Label
	err := s.pool.QueryRow(ctx, `
		INSERT INTO labels (name, color) VALUES ($1, $2)
		RETURNING id::text, name, color, created_at
	`, in.Name, color).Scan(&l.ID, &l.Name, &l.Color, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create label: %w", err)
	}
	return &l, nil
}

// UpdateLabel applies a partial update to a label. Returns (nil, nil) if not found.
func (s *PostgresStore) UpdateLabel(ctx context.Context, id string, in models.UpdateLabelInput) (*models.Label, error) {
	sets := []string{}
	args := []any{}
	n := 1

	if in.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", n))
		args = append(args, *in.Name)
		n++
	}
	if in.Color != nil {
		sets = append(sets, fmt.Sprintf("color = $%d", n))
		args = append(args, *in.Color)
		n++
	}
	if len(sets) == 0 {
		// Nothing to update; return the current record.
		var l models.Label
		err := s.pool.QueryRow(ctx, `SELECT id::text, name, color, created_at FROM labels WHERE id = $1::uuid`, id).
			Scan(&l.ID, &l.Name, &l.Color, &l.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return &l, err
	}

	args = append(args, id)
	var l models.Label
	err := s.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE labels SET %s WHERE id = $%d::uuid
		RETURNING id::text, name, color, created_at
	`, strings.Join(sets, ", "), n), args...).Scan(&l.ID, &l.Name, &l.Color, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update label: %w", err)
	}
	return &l, nil
}

// DeleteLabel removes a label and its task associations (via ON DELETE CASCADE).
func (s *PostgresStore) DeleteLabel(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM labels WHERE id = $1::uuid", id)
	if err != nil {
		return fmt.Errorf("delete label: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Tasks ────────────────────────────────────────────────────────────────

// taskCols is the common column list used in all task SELECT statements.
// UUIDs are cast to text for easy scanning into *string / string.
const taskCols = `
	t.id::text,
	t.title,
	t.notes,
	t.project_id::text,
	t.parent_id::text,
	t.priority,
	to_char(t.due_date, 'YYYY-MM-DD'),
	t.completed,
	t.completed_at,
	t.position,
	t.created_at,
	t.updated_at`

// scanTask reads one task row into a models.Task. The caller must call
// rows.Next() before calling this function. Labels are not populated here;
// use loadLabels after collecting all tasks.
func scanTask(rows pgx.Rows) (models.Task, error) {
	var t models.Task
	t.Labels = make([]models.Label, 0)
	err := rows.Scan(
		&t.ID, &t.Title, &t.Notes,
		&t.ProjectID, &t.ParentID,
		&t.Priority, &t.DueDate,
		&t.Completed, &t.CompletedAt,
		&t.Position, &t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

// loadLabels bulk-fetches all labels for the supplied tasks (single round-trip)
// and attaches them in place.
func (s *PostgresStore) loadLabels(ctx context.Context, tasks []models.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	ids := make([]string, len(tasks))
	index := make(map[string]int, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
		index[t.ID] = i
	}

	rows, err := s.pool.Query(ctx, `
		SELECT tl.task_id::text, l.id::text, l.name, l.color, l.created_at
		FROM task_labels tl
		JOIN labels l ON l.id = tl.label_id
		WHERE tl.task_id::text = ANY($1::text[])
		ORDER BY l.name
	`, ids)
	if err != nil {
		return fmt.Errorf("load labels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var taskID string
		var l models.Label
		if err := rows.Scan(&taskID, &l.ID, &l.Name, &l.Color, &l.CreatedAt); err != nil {
			return err
		}
		if i, ok := index[taskID]; ok {
			tasks[i].Labels = append(tasks[i].Labels, l)
		}
	}
	return rows.Err()
}

// ListTasks returns top-level tasks matching the filter, ordered by position
// then created_at. Labels are loaded in a second query.
func (s *PostgresStore) ListTasks(ctx context.Context, f models.TaskFilter) ([]models.Task, error) {
	q := "SELECT" + taskCols + " FROM tasks t WHERE 1=1"
	args := []any{}
	n := 1

	// Subtask vs top-level filtering
	if f.ParentID != nil {
		q += fmt.Sprintf(" AND t.parent_id = $%d::uuid", n)
		args = append(args, *f.ParentID)
		n++
	} else {
		q += " AND t.parent_id IS NULL"
	}

	if f.ProjectID != nil {
		q += fmt.Sprintf(" AND t.project_id = $%d::uuid", n)
		args = append(args, *f.ProjectID)
		n++
	}

	if f.DueToday {
		q += " AND t.due_date = CURRENT_DATE"
	} else if f.Upcoming {
		q += " AND t.due_date >= CURRENT_DATE AND t.due_date <= CURRENT_DATE + INTERVAL '7 days'"
	}

	if f.Completed != nil {
		q += fmt.Sprintf(" AND t.completed = $%d", n)
		args = append(args, *f.Completed)
		n++
	}

	if f.Search != "" {
		q += fmt.Sprintf(" AND (t.title ILIKE $%d OR t.notes ILIKE $%d)", n, n)
		args = append(args, "%"+f.Search+"%")
		n++
	}
	_ = n

	q += " ORDER BY t.position, t.created_at"

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]models.Task, 0)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.loadLabels(ctx, tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetTask fetches a single task by ID with its labels and subtasks populated.
// Returns (nil, nil) if not found.
func (s *PostgresStore) GetTask(ctx context.Context, id string) (*models.Task, error) {
	rows, err := s.pool.Query(ctx, "SELECT"+taskCols+" FROM tasks t WHERE t.id = $1::uuid", id)
	if err != nil {
		return nil, fmt.Errorf("get task query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	t, err := scanTask(rows)
	if err != nil {
		return nil, fmt.Errorf("scan task: %w", err)
	}
	rows.Close()

	// Load labels for this single task.
	slice := []models.Task{t}
	if err := s.loadLabels(ctx, slice); err != nil {
		return nil, err
	}
	t = slice[0]

	// Load subtasks recursively one level deep.
	sub, err := s.ListTasks(ctx, models.TaskFilter{ParentID: &t.ID})
	if err != nil {
		return nil, err
	}
	t.Subtasks = sub
	return &t, nil
}

// CreateTask inserts a new task and attaches any specified labels atomically.
func (s *PostgresStore) CreateTask(ctx context.Context, in models.CreateTaskInput) (*models.Task, error) {
	priority := in.Priority
	if priority < 1 || priority > 4 {
		priority = 4
	}

	// Default to Inbox when no project is supplied.
	pid := in.ProjectID
	if pid == nil || *pid == "" {
		inbox := inboxProjectID
		pid = &inbox
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var newID string
	err = tx.QueryRow(ctx, `
		INSERT INTO tasks (title, notes, project_id, parent_id, priority, due_date, position)
		VALUES (
			$1, $2,
			$3::uuid,
			$4::uuid,
			$5,
			NULLIF($6, '')::date,
			COALESCE((
				SELECT MAX(position) FROM tasks
				WHERE project_id IS NOT DISTINCT FROM $3::uuid
				  AND parent_id  IS NOT DISTINCT FROM $4::uuid
			), 0) + 1
		)
		RETURNING id::text
	`, in.Title, in.Notes, pid, in.ParentID, priority, nullableStr(in.DueDate)).Scan(&newID)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}

	if err := attachLabels(ctx, tx, newID, in.LabelIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return s.GetTask(ctx, newID)
}

// UpdateTask applies a partial update to a task and optionally replaces labels.
func (s *PostgresStore) UpdateTask(ctx context.Context, id string, in models.UpdateTaskInput) (*models.Task, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	n := 1

	if in.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", n))
		args = append(args, *in.Title)
		n++
	}
	if in.Notes != nil {
		sets = append(sets, fmt.Sprintf("notes = $%d", n))
		args = append(args, *in.Notes)
		n++
	}
	if in.ProjectID != nil {
		if *in.ProjectID == "" {
			sets = append(sets, fmt.Sprintf("project_id = $%d::uuid", n))
			args = append(args, inboxProjectID)
		} else {
			sets = append(sets, fmt.Sprintf("project_id = $%d::uuid", n))
			args = append(args, *in.ProjectID)
		}
		n++
	}
	if in.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", n))
		args = append(args, *in.Priority)
		n++
	}
	if in.Completed != nil {
		sets = append(sets, fmt.Sprintf("completed = $%d", n))
		args = append(args, *in.Completed)
		n++
		if *in.Completed {
			sets = append(sets, "completed_at = NOW()")
		} else {
			sets = append(sets, "completed_at = NULL")
		}
	}
	if in.DueDate != nil {
		if *in.DueDate == "" {
			sets = append(sets, "due_date = NULL")
		} else {
			sets = append(sets, fmt.Sprintf("due_date = $%d::date", n))
			args = append(args, *in.DueDate)
			n++
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	args = append(args, id)
	_, err = tx.Exec(ctx,
		fmt.Sprintf("UPDATE tasks SET %s WHERE id = $%d::uuid", strings.Join(sets, ", "), n),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	// Replace labels only when the field is explicitly provided.
	if in.LabelIDs != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM task_labels WHERE task_id = $1::uuid", id); err != nil {
			return nil, err
		}
		if err := attachLabels(ctx, tx, id, in.LabelIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return s.GetTask(ctx, id)
}

// DeleteTask removes a task and all its subtasks (ON DELETE CASCADE).
func (s *PostgresStore) DeleteTask(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM tasks WHERE id = $1::uuid", id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SearchTasks performs a case-insensitive search across title and notes.
func (s *PostgresStore) SearchTasks(ctx context.Context, query string) ([]models.Task, error) {
	return s.ListTasks(ctx, models.TaskFilter{Search: query})
}

// ─── Internal helpers ─────────────────────────────────────────────────────

// attachLabels inserts task_labels rows inside the given transaction.
func attachLabels(ctx context.Context, tx pgx.Tx, taskID string, labelIDs []string) error {
	for _, lid := range labelIDs {
		if lid == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO task_labels (task_id, label_id)
			VALUES ($1::uuid, $2::uuid)
			ON CONFLICT DO NOTHING
		`, taskID, lid); err != nil {
			return fmt.Errorf("attach label %s: %w", lid, err)
		}
	}
	return nil
}

// nullableStr returns nil if p is nil or *p is empty, otherwise *p.
// Used so that an empty DueDate string is stored as a SQL NULL.
func nullableStr(p *string) any {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}
