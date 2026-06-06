package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phoenixha4/slate/internal/handlers"
	"github.com/phoenixha4/slate/internal/models"
	"github.com/phoenixha4/slate/internal/store"
)

// ─── Mock store ───────────────────────────────────────────────────────────
// mockStore is a minimal in-memory implementation of store.Store used only
// in tests. Methods not needed by the tests under test return zero values.

type mockStore struct {
	tasks    []models.Task
	projects []models.Project
	labels   []models.Label
}

func newMock() *mockStore {
	inbox := "00000000-0000-0000-0000-000000000001"
	return &mockStore{
		projects: []models.Project{{ID: inbox, Name: "Inbox", Color: "#7c6af7", Icon: "inbox"}},
		labels:   []models.Label{},
		tasks:    []models.Task{},
	}
}

func (m *mockStore) Ping(_ context.Context) error                             { return nil }
func (m *mockStore) ListProjects(_ context.Context) ([]models.Project, error) { return m.projects, nil }
func (m *mockStore) GetProject(_ context.Context, id string) (*models.Project, error) {
	for _, p := range m.projects {
		if p.ID == id {
			cp := p
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *mockStore) CreateProject(_ context.Context, in models.CreateProjectInput) (*models.Project, error) {
	p := models.Project{ID: "new-proj", Name: in.Name, Color: in.Color, Icon: in.Icon}
	m.projects = append(m.projects, p)
	return &p, nil
}
func (m *mockStore) UpdateProject(_ context.Context, id string, in models.UpdateProjectInput) (*models.Project, error) {
	for i := range m.projects {
		if m.projects[i].ID == id {
			if in.Name != nil {
				m.projects[i].Name = *in.Name
			}
			cp := m.projects[i]
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *mockStore) DeleteProject(_ context.Context, id string) error { return nil }

func (m *mockStore) ListLabels(_ context.Context) ([]models.Label, error) { return m.labels, nil }
func (m *mockStore) CreateLabel(_ context.Context, in models.CreateLabelInput) (*models.Label, error) {
	l := models.Label{ID: "new-label", Name: in.Name, Color: in.Color}
	m.labels = append(m.labels, l)
	return &l, nil
}
func (m *mockStore) UpdateLabel(_ context.Context, id string, in models.UpdateLabelInput) (*models.Label, error) {
	return nil, nil
}
func (m *mockStore) DeleteLabel(_ context.Context, id string) error { return nil }

func (m *mockStore) ListTasks(_ context.Context, f models.TaskFilter) ([]models.Task, error) {
	result := make([]models.Task, 0)
	for _, t := range m.tasks {
		if f.Completed != nil && t.Completed != *f.Completed {
			continue
		}
		if f.ProjectID != nil && (t.ProjectID == nil || *t.ProjectID != *f.ProjectID) {
			continue
		}
		result = append(result, t)
	}
	return result, nil
}
func (m *mockStore) GetTask(_ context.Context, id string) (*models.Task, error) {
	for _, t := range m.tasks {
		if t.ID == id {
			cp := t
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *mockStore) CreateTask(_ context.Context, in models.CreateTaskInput) (*models.Task, error) {
	t := models.Task{
		ID:     "new-task",
		Title:  in.Title,
		Notes:  in.Notes,
		Labels: []models.Label{},
	}
	m.tasks = append(m.tasks, t)
	return &t, nil
}
func (m *mockStore) UpdateTask(_ context.Context, id string, in models.UpdateTaskInput) (*models.Task, error) {
	for i := range m.tasks {
		if m.tasks[i].ID == id {
			if in.Title != nil {
				m.tasks[i].Title = *in.Title
			}
			if in.Completed != nil {
				m.tasks[i].Completed = *in.Completed
			}
			cp := m.tasks[i]
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *mockStore) DeleteTask(_ context.Context, id string) error {
	for i, t := range m.tasks {
		if t.ID == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			return nil
		}
	}
	return store.ErrNotFound
}
func (m *mockStore) SearchTasks(_ context.Context, q string) ([]models.Task, error) {
	result := make([]models.Task, 0)
	for _, t := range m.tasks {
		if len(q) > 0 && (len(t.Title) >= len(q) && t.Title[:len(q)] == q) {
			result = append(result, t)
		}
	}
	return result, nil
}

// ─── Tests ────────────────────────────────────────────────────────────────

func TestListTasks_ReturnsEmptySlice(t *testing.T) {
	h := handlers.NewHandler(newMock(), nil)
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w := httptest.NewRecorder()

	h.ListTasks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var tasks []models.Task
	if err := json.Unmarshal(w.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tasks == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestCreateTask_RequiresTitle(t *testing.T) {
	h := handlers.NewHandler(newMock(), nil)
	body, _ := json.Marshal(map[string]string{"title": ""})
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateTask(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateAndGetTask(t *testing.T) {
	m := newMock()
	h := handlers.NewHandler(m, nil)

	// Create
	body, _ := json.Marshal(models.CreateTaskInput{Title: "Buy milk", Priority: 3})
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateTask(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created models.Task
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.Title != "Buy milk" {
		t.Fatalf("unexpected title: %s", created.Title)
	}
}

func TestUpdateTask_ToggleComplete(t *testing.T) {
	m := newMock()
	m.tasks = []models.Task{{ID: "t1", Title: "Test task", Labels: []models.Label{}}}
	h := handlers.NewHandler(m, nil)

	done := true
	body, _ := json.Marshal(models.UpdateTaskInput{Completed: &done})
	req := httptest.NewRequest(http.MethodPatch, "/tasks/t1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "t1")
	w := httptest.NewRecorder()

	h.UpdateTask(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updated models.Task
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !updated.Completed {
		t.Fatal("expected task to be completed")
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	h := handlers.NewHandler(newMock(), nil)
	req := httptest.NewRequest(http.MethodDelete, "/tasks/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	h.DeleteTask(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestListProjects_AlwaysHasInbox(t *testing.T) {
	h := handlers.NewHandler(newMock(), nil)
	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	w := httptest.NewRecorder()

	h.ListProjects(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var projects []models.Project
	if err := json.Unmarshal(w.Body.Bytes(), &projects); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(projects) == 0 {
		t.Fatal("expected at least the Inbox project")
	}
}
