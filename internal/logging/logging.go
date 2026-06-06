// Package logging provides application log handlers.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// PrettyHandler formats slog records for humans reading local development logs.
// It keeps the structured fields, but presents them as a compact console line
// instead of JSON or slog's default text format.
type PrettyHandler struct {
	out    io.Writer
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
	mu     *sync.Mutex
}

// NewPrettyHandler returns a dependency-free, development-friendly slog handler.
func NewPrettyHandler(out io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	var level slog.Leveler = slog.LevelInfo
	if opts != nil && opts.Level != nil {
		level = opts.Level
	}
	return &PrettyHandler{
		out:   out,
		level: level,
		mu:    &sync.Mutex{},
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Time.Format("15:04:05"))
	b.WriteByte(' ')
	b.WriteString(formatLevel(r.Level))
	b.WriteByte(' ')
	b.WriteString(r.Message)

	writeAttrs(&b, h.attrs, h.groups)
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&b, a, h.groups)
		return true
	})
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, b.String())
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := h.clone()
	next.attrs = append(next.attrs, attrs...)
	return next
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := h.clone()
	next.groups = append(next.groups, name)
	return next
}

func (h *PrettyHandler) clone() *PrettyHandler {
	next := *h
	next.attrs = append([]slog.Attr(nil), h.attrs...)
	next.groups = append([]string(nil), h.groups...)
	return &next
}

func writeAttrs(b *strings.Builder, attrs []slog.Attr, groups []string) {
	for _, a := range attrs {
		writeAttr(b, a, groups)
	}
}

func writeAttr(b *strings.Builder, a slog.Attr, groups []string) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}

	key := a.Key
	if len(groups) > 0 {
		key = strings.Join(append(append([]string{}, groups...), key), ".")
	}

	if a.Value.Kind() == slog.KindGroup {
		writeAttrs(b, a.Value.Group(), append(groups, a.Key))
		return
	}

	b.WriteString("  ")
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(formatValue(a.Value))
}

func formatLevel(level slog.Level) string {
	switch {
	case level <= slog.LevelDebug:
		return "\x1b[36mDEBUG\x1b[0m"
	case level < slog.LevelWarn:
		return "\x1b[32mINFO \x1b[0m"
	case level < slog.LevelError:
		return "\x1b[33mWARN \x1b[0m"
	default:
		return "\x1b[31mERROR\x1b[0m"
	}
}

func formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		if s == "" || strings.ContainsAny(s, " \t\n\r") {
			return fmt.Sprintf("%q", s)
		}
		return s
	case slog.KindDuration:
		return v.Duration().Round(time.Millisecond).String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	default:
		return fmt.Sprint(v.Any())
	}
}
