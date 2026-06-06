// Package models defines the domain types shared across all application layers.
package models

import "time"

// Project is a named collection of tasks with a colour and icon.
type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	Icon      string    `json:"icon"`
	Position  int       `json:"position"`
	Archived  bool      `json:"archived"`
	TaskCount int       `json:"task_count"` // active (non-completed) top-level tasks
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Label is a coloured tag that can be attached to any number of tasks.
type Label struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// Task is the core entity. It supports subtasks (ParentID), project
// membership, a four-level priority (1 = urgent, 4 = none), an optional
// due date, and zero or more labels.
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Notes       string     `json:"notes"`
	ProjectID   *string    `json:"project_id"`
	ParentID    *string    `json:"parent_id"`
	Priority    int        `json:"priority"`    // 1–4
	DueDate     *string    `json:"due_date"`    // "YYYY-MM-DD" or null
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`
	Position    int        `json:"position"`
	Labels      []Label    `json:"labels"`
	Subtasks    []Task     `json:"subtasks,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ─── Input / request types ────────────────────────────────────────────────

// CreateProjectInput is the body accepted by POST /projects.
type CreateProjectInput struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
}

// UpdateProjectInput is the body accepted by PATCH /projects/:id.
// Every field is a pointer so that callers only send the fields they want
// to change (partial update / JSON merge-patch style).
type UpdateProjectInput struct {
	Name     *string `json:"name"`
	Color    *string `json:"color"`
	Icon     *string `json:"icon"`
	Archived *bool   `json:"archived"`
}

// CreateLabelInput is the body accepted by POST /labels.
type CreateLabelInput struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// UpdateLabelInput is the body accepted by PATCH /labels/:id.
type UpdateLabelInput struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// CreateTaskInput is the body accepted by POST /tasks.
type CreateTaskInput struct {
	Title     string   `json:"title"`
	Notes     string   `json:"notes"`
	ProjectID *string  `json:"project_id"`
	ParentID  *string  `json:"parent_id"`
	Priority  int      `json:"priority"`
	DueDate   *string  `json:"due_date"` // "YYYY-MM-DD"
	LabelIDs  []string `json:"label_ids"`
}

// UpdateTaskInput is the body accepted by PATCH /tasks/:id.
type UpdateTaskInput struct {
	Title     *string  `json:"title"`
	Notes     *string  `json:"notes"`
	ProjectID *string  `json:"project_id"`
	Priority  *int     `json:"priority"`
	DueDate   *string  `json:"due_date"` // "YYYY-MM-DD" or "" to clear
	Completed *bool    `json:"completed"`
	LabelIDs  []string `json:"label_ids"` // replaces existing labels when present
}

// TaskFilter describes the optional constraints passed to ListTasks.
type TaskFilter struct {
	ProjectID *string // restrict to a single project
	ParentID  *string // restrict to subtasks of this parent
	DueToday  bool    // due_date = today
	Upcoming  bool    // due_date in the next 7 days (inclusive of today)
	Completed *bool   // nil = all, true = done, false = active
	Search    string  // ILIKE match against title and notes
}
