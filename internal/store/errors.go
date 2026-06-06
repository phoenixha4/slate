package store

import "errors"

var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrInboxProtected is returned when the caller attempts to delete the
	// built-in Inbox project.
	ErrInboxProtected = errors.New("inbox project cannot be deleted")
)
