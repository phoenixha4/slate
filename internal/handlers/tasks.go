package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/phoenixha4/slate/internal/models"
	"github.com/phoenixha4/slate/internal/store"
)

// ListTasks handles GET /tasks.
// Query params: project_id, due=today|upcoming, completed=true|false
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := models.TaskFilter{}

	if pid := q.Get("project_id"); pid != "" {
		f.ProjectID = &pid
	}
	switch q.Get("due") {
	case "today":
		f.DueToday = true
	case "upcoming":
		f.Upcoming = true
	}
	switch q.Get("completed") {
	case "true":
		t := true
		f.Completed = &t
	case "false":
		t := false
		f.Completed = &t
	}

	tasks, err := h.store.ListTasks(r.Context(), f)
	if err != nil {
		h.log.Error("list tasks", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

// GetTask handles GET /tasks/{id}.
// Returns the task with its labels and one level of subtasks.
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.store.GetTask(r.Context(), id)
	if err != nil {
		h.log.Error("get task", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// SearchTasks handles GET /tasks/search?q=<query>.
func (h *Handler) SearchTasks(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, []models.Task{})
		return
	}
	tasks, err := h.store.SearchTasks(r.Context(), query)
	if err != nil {
		h.log.Error("search tasks", "query", query, "error", err)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

// CreateTask handles POST /tasks.
// Body: CreateTaskInput JSON.
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var in models.CreateTaskInput
	if !decodeBody(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	t, err := h.store.CreateTask(r.Context(), in)
	if err != nil {
		h.log.Error("create task", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// UpdateTask handles PATCH /tasks/{id}.
// Body: UpdateTaskInput JSON — only supplied fields are updated.
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in models.UpdateTaskInput
	if !decodeBody(w, r, &in) {
		return
	}
	t, err := h.store.UpdateTask(r.Context(), id, in)
	if err != nil {
		h.log.Error("update task", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// DeleteTask handles DELETE /tasks/{id}.
// Deletes the task and all its subtasks (ON DELETE CASCADE).
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := h.store.DeleteTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		h.log.Error("delete task", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
