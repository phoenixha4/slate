// Package assets embeds the compiled frontend files into the Go binary.
// The "all:" prefix ensures dotfiles (e.g. .gitkeep) are also included
// if present; it has no effect on normal directories.
package assets

import "embed"

//go:embed all:frontend
var FS embed.FS
