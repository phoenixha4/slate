package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/phoenixha4/learning_go/internal/models"
	"github.com/phoenixha4/learning_go/internal/store"
)

// ListLabels handles GET /labels.
func (h *Handler) ListLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := h.store.ListLabels(r.Context())
	if err != nil {
		h.log.Error("list labels", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list labels")
		return
	}
	writeJSON(w, http.StatusOK, labels)
}

// CreateLabel handles POST /labels.
// Body: {"name":"string","color":"#rrggbb"}
func (h *Handler) CreateLabel(w http.ResponseWriter, r *http.Request) {
	var in models.CreateLabelInput
	if !decodeBody(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	l, err := h.store.CreateLabel(r.Context(), in)
	if err != nil {
		h.log.Error("create label", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create label")
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

// UpdateLabel handles PATCH /labels/{id}.
func (h *Handler) UpdateLabel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in models.UpdateLabelInput
	if !decodeBody(w, r, &in) {
		return
	}
	l, err := h.store.UpdateLabel(r.Context(), id, in)
	if err != nil {
		h.log.Error("update label", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update label")
		return
	}
	if l == nil {
		writeError(w, http.StatusNotFound, "label not found")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// DeleteLabel handles DELETE /labels/{id}.
func (h *Handler) DeleteLabel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := h.store.DeleteLabel(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "label not found")
			return
		}
		h.log.Error("delete label", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete label")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
