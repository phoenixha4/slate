// Package store defines the data-access interface used by HTTP handlers.
// Accepting an interface rather than a concrete type makes handlers trivially
// testable: tests can supply an in-memory mock without needing a real database.
package store

import (
	"context"

	"github.com/phoenixha4/learning_go/internal/models"
)

// Store is the single data-access interface for the whole application.
type Store interface {
	Ping(ctx context.Context) error

	// ─── Projects ─────────────────────────────────────────────────────────
	ListProjects(ctx context.Context) ([]models.Project, error)
	GetProject(ctx context.Context, id string) (*models.Project, error)
	CreateProject(ctx context.Context, in models.CreateProjectInput) (*models.Project, error)
	UpdateProject(ctx context.Context, id string, in models.UpdateProjectInput) (*models.Project, error)
	DeleteProject(ctx context.Context, id string) error

	// ─── Labels ───────────────────────────────────────────────────────────
	ListLabels(ctx context.Context) ([]models.Label, error)
	CreateLabel(ctx context.Context, in models.CreateLabelInput) (*models.Label, error)
	UpdateLabel(ctx context.Context, id string, in models.UpdateLabelInput) (*models.Label, error)
	DeleteLabel(ctx context.Context, id string) error

	// ─── Tasks ────────────────────────────────────────────────────────────
	ListTasks(ctx context.Context, f models.TaskFilter) ([]models.Task, error)
	GetTask(ctx context.Context, id string) (*models.Task, error)
	CreateTask(ctx context.Context, in models.CreateTaskInput) (*models.Task, error)
	UpdateTask(ctx context.Context, id string, in models.UpdateTaskInput) (*models.Task, error)
	DeleteTask(ctx context.Context, id string) error
	SearchTasks(ctx context.Context, query string) ([]models.Task, error)
}
