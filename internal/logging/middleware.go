package logging

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the status code and the
// number of bytes written, so request logs can report them after completion.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Flush forwards to the underlying writer if it supports flushing.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// HTTPMiddleware logs every request once it finishes, reporting the method,
// path, status code, byte count, and duration.
func HTTPMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	// Detect color once from the handler so request logs match the logger.
	colored := isColorLogger(logger)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}

			defer func() {
				if rv := recover(); rv != nil {
					if rec.status == 0 {
						rec.WriteHeader(http.StatusInternalServerError)
					}
					logger.Error("panic recovered while serving request",
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.Any("panic", rv),
					)
				}

				method := colorize(methodColor(r.Method), r.Method, colored)
				path := colorize(colUnderline, r.URL.Path, colored)

				logger.LogAttrs(r.Context(), statusLevel(rec.status),
					fmt.Sprintf("%s %s", method, path),
					slog.Int("status", rec.status),
					slog.Int("bytes", rec.bytes),
					slog.String("remote", r.RemoteAddr),
					slog.Duration("duration", time.Since(start)),
				)
			}()

			next.ServeHTTP(rec, r)
		})
	}
}

// statusLevel maps an HTTP status to a log level so 5xx surfaces as errors.
func statusLevel(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// colorize wraps s in an ANSI color when enabled.
func colorize(color, s string, enabled bool) string {
	if !enabled {
		return s
	}
	return color + s + colReset
}

// isColorLogger reports whether the logger's handler is a color-enabled
// prettyHandler, so middleware output matches the rest of the logs.
func isColorLogger(logger *slog.Logger) bool {
	if logger == nil {
		return false
	}
	if h, ok := logger.Handler().(*prettyHandler); ok {
		return h.color
	}
	return false
}
