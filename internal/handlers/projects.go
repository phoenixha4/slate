package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/phoenixha4/slate/internal/models"
	"github.com/phoenixha4/slate/internal/store"
)

// ListProjects handles GET /projects.
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ListProjects(r.Context())
	if err != nil {
		h.log.Error("list projects", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

// GetProject handles GET /projects/{id}.
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.store.GetProject(r.Context(), id)
	if err != nil {
		h.log.Error("get project", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get project")
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// CreateProject handles POST /projects.
// Body: {"name":"string","color":"#rrggbb","icon":"string"}
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var in models.CreateProjectInput
	if !decodeBody(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	p, err := h.store.CreateProject(r.Context(), in)
	if err != nil {
		h.log.Error("create project", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// UpdateProject handles PATCH /projects/{id}.
// Accepts any subset of project fields (partial update).
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in models.UpdateProjectInput
	if !decodeBody(w, r, &in) {
		return
	}
	p, err := h.store.UpdateProject(r.Context(), id, in)
	if err != nil {
		h.log.Error("update project", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// DeleteProject handles DELETE /projects/{id}.
// The built-in Inbox project returns 403.
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := h.store.DeleteProject(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrInboxProtected) {
			writeError(w, http.StatusForbidden, "the Inbox project cannot be deleted")
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		h.log.Error("delete project", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
