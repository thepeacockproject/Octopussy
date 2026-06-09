// Package logging provides a small, dependency-light structured logger built
// on top of log/slog. It renders nicely-colored, human-readable lines to the
// console (with a [timestamp] [LEVEL] message key=value layout) and is
// inspired by Peacock's logging interop layer.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ANSI color codes used by the pretty handler. They are stripped automatically
// when color is disabled (non-TTY output or the NO_COLOR env var being set).
const (
	colReset     = "\x1b[0m"
	colGray      = "\x1b[90m"
	colRed       = "\x1b[31m"
	colYellow    = "\x1b[33m"
	colBlue      = "\x1b[34m"
	colCyan      = "\x1b[36m"
	colGreen     = "\x1b[32m"
	colMagenta   = "\x1b[35m"
	colUnderline = "\x1b[4m"
)

// Options configures a new logger.
type Options struct {
	// Level is the minimum level that will be emitted. Defaults to LevelInfo.
	Level slog.Level
	// Writer is where log lines are written. Defaults to os.Stdout.
	Writers []io.Writer
}

func New(opts Options) *slog.Logger {
	writers := []io.Writer{os.Stdout}
	writers = append(writers, opts.Writers...)

	w := io.MultiWriter(writers...)

	return slog.New(&prettyHandler{
		w:     w,
		mu:    &sync.Mutex{},
		level: opts.Level,
	})
}

// prettyHandler is a slog.Handler that emits colorized, single-line records.
type prettyHandler struct {
	w     io.Writer
	mu    *sync.Mutex
	level slog.Level
	color bool

	// attrs and group accumulated from With* calls.
	attrs  []slog.Attr
	groups []string
}

func (h *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder

	ts := r.Time
	if ts.IsZero() {
		ts = time.Now()
	}
	stamp := ts.Format("15:04:05.000")

	b.WriteString(h.paint(colGray, "["+stamp+"]"))
	b.WriteByte(' ')
	b.WriteString("[" + h.levelLabel(r.Level) + "]")
	b.WriteByte(' ')
	b.WriteString(r.Message)

	// Collect attributes: handler-scoped first, then record-scoped.
	attrs := make([]slog.Attr, 0, len(h.attrs)+r.NumAttrs())
	attrs = append(attrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	for _, a := range attrs {
		h.appendAttr(&b, h.groupPrefix(), a)
	}

	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

func (h *prettyHandler) groupPrefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	return strings.Join(h.groups, ".") + "."
}

// appendAttr renders a single attribute as " key=value", recursing into groups.
func (h *prettyHandler) appendAttr(b *strings.Builder, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}

	if a.Value.Kind() == slog.KindGroup {
		group := a.Value.Group()
		if len(group) == 0 {
			return
		}
		newPrefix := prefix
		if a.Key != "" {
			newPrefix = prefix + a.Key + "."
		}
		// Stable ordering keeps output deterministic and easy to scan.
		sort.SliceStable(group, func(i, j int) bool { return group[i].Key < group[j].Key })
		for _, g := range group {
			h.appendAttr(b, newPrefix, g)
		}
		return
	}

	b.WriteByte(' ')
	b.WriteString(h.paint(colCyan, prefix+a.Key))
	b.WriteByte('=')
	b.WriteString(quoteValue(a.Value.String()))
}

// quoteValue wraps a value in quotes when it contains spaces.
func quoteValue(v string) string {
	if strings.ContainsAny(v, " \t\n") {
		return strconv.Quote(v)
	}
	return v
}

func (h *prettyHandler) levelLabel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return h.paint(colRed, "ERROR")
	case level >= slog.LevelWarn:
		return h.paint(colYellow, "WARN")
	case level >= slog.LevelInfo:
		return h.paint(colBlue, "INFO")
	default:
		return h.paint(colCyan, "DEBUG")
	}
}

// paint wraps s in an ANSI color when color output is enabled.
func (h *prettyHandler) paint(color, s string) string {
	return color + s + colReset
}

// methodColor maps an HTTP method to a color for request logs.
func methodColor(method string) string {
	switch method {
	case "GET":
		return colGreen
	case "POST":
		return colYellow
	case "PUT", "PATCH":
		return colBlue
	case "DELETE":
		return colRed
	default:
		return colMagenta
	}
}
