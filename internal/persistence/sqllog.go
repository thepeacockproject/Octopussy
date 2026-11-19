package persistence

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jinzhu/inflection"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// sqlLogger is a gorm logger.Interface that renders queries the way Rails'
// ActiveRecord does, routed through the app's slog logger:
//
//	CustomProfileData Load (0.3ms)  SELECT * FROM `custom_profile_data` WHERE ...
//	  ↳ internal/controller/customprofiledata.go:42
//
// Every query is logged at Debug when the log mode is Info. Slow queries
// surface at Warn and failures at Error regardless.
type sqlLogger struct {
	log           *slog.Logger
	level         logger.LogLevel
	slowThreshold time.Duration
	color         bool
}

func newSQLLogger(log *slog.Logger, level logger.LogLevel) *sqlLogger {
	if log == nil {
		log = slog.Default()
	}
	_, noColor := os.LookupEnv("NO_COLOR")
	return &sqlLogger{
		log:           log,
		level:         level,
		slowThreshold: 200 * time.Millisecond,
		color:         !noColor,
	}
}

func (l *sqlLogger) LogMode(level logger.LogLevel) logger.Interface {
	clone := *l
	clone.level = level
	return &clone
}

func (l *sqlLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.level >= logger.Info {
		l.log.InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *sqlLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.level >= logger.Warn {
		l.log.WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *sqlLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.level >= logger.Error {
		l.log.ErrorContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *sqlLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)

	failed := err != nil && !errors.Is(err, gorm.ErrRecordNotFound)
	slow := elapsed > l.slowThreshold

	switch {
	case failed && l.level >= logger.Error:
		sql, _ := fc()
		l.log.ErrorContext(ctx, l.format(ctx, sql, elapsed), slog.String("error", err.Error()))
	case slow && l.level >= logger.Warn:
		sql, _ := fc()
		l.log.WarnContext(ctx, l.format(ctx, sql, elapsed), slog.String("slow", ">"+l.slowThreshold.String()))
	case l.level >= logger.Info:
		sql, _ := fc()
		l.log.DebugContext(ctx, l.format(ctx, sql, elapsed))
	}
}

// format builds the Rails-style "Name (1.2ms)  SQL\n  ↳ file:line" message.
func (l *sqlLogger) format(ctx context.Context, sql string, elapsed time.Duration) string {
	name := fmt.Sprintf("%s (%.1fms)", queryName(ctx, sql), float64(elapsed.Microseconds())/1000)

	var b strings.Builder
	b.WriteString(l.paint(ansiBold, name))
	b.WriteString("  ")
	b.WriteString(l.paint(sqlColor(sql), sql))
	if src := appCaller(); src != "" {
		b.WriteString("\n  ↳ ")
		b.WriteString(l.paint(ansiGray, src))
	}
	return b.String()
}

func (l *sqlLogger) paint(code, s string) string {
	if !l.color {
		return s
	}
	return code + s + ansiReset
}

const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiGray    = "\x1b[90m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiBlue    = "\x1b[34m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
	ansiWhite   = "\x1b[37m"
)

var (
	reForUpdate = regexp.MustCompile(`(?is)^\s*select .*for update`)
	reTxEnd     = regexp.MustCompile(`(?i)transaction\s*$`)
	reTable     = regexp.MustCompile("(?i)\\b(?:from|into|update)\\s+([`\"\\[]?[\\w.]+[`\"\\]]?)")
)

// sqlColor mirrors ActiveRecord::LogSubscriber#sql_color.
func sqlColor(sql string) string {
	verb := strings.ToUpper(firstWord(sql))
	switch {
	case verb == "ROLLBACK":
		return ansiRed
	case reForUpdate.MatchString(sql), verb == "LOCK":
		return ansiWhite
	case verb == "SELECT":
		return ansiBlue
	case verb == "INSERT":
		return ansiGreen
	case verb == "UPDATE":
		return ansiYellow
	case verb == "DELETE":
		return ansiRed
	case reTxEnd.MatchString(sql):
		return ansiCyan
	default:
		return ansiMagenta
	}
}

// queryName produces the Rails-style label: "<Model> <Action>", "SCHEMA" for
// catalog introspection, or "SQL" for DDL and anything unrecognised.
func queryName(ctx context.Context, sql string) string {
	var action string
	switch strings.ToUpper(firstWord(sql)) {
	case "SELECT":
		action = "Load"
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "SELECT COUNT(") {
			action = "Count"
		}
	case "INSERT":
		action = "Create"
	case "UPDATE":
		action = "Update"
	case "DELETE":
		action = "Destroy"
	case "BEGIN", "COMMIT", "ROLLBACK", "SAVEPOINT", "RELEASE":
		return "TRANSACTION"
	default:
		return "SQL"
	}

	table := ""
	if m := reTable.FindStringSubmatch(sql); m != nil {
		table = strings.Trim(m[1], "`\"[]")
		if i := strings.LastIndexByte(table, '.'); i >= 0 {
			table = table[i+1:]
		}
	}
	if isCatalogTable(table) {
		return "SCHEMA"
	}

	model, _ := ctx.Value(modelNameKey{}).(string)
	if model == "" && table != "" {
		model = camelize(inflection.Singular(table))
	}
	if model == "" {
		return "SQL"
	}
	return model + " " + action
}

func isCatalogTable(table string) bool {
	t := strings.ToLower(table)
	return strings.HasPrefix(t, "sqlite_") || strings.HasPrefix(t, "pg_") || t == "tables" || t == "columns"
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexFunc(s, func(r rune) bool { return r == ' ' || r == '\n' || r == '\t' || r == '(' }); i >= 0 {
		return s[:i]
	}
	return s
}

func camelize(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// modelNameKey carries the statement's Go model name from a gorm callback into
// Trace, which otherwise only sees the rendered SQL.
type modelNameKey struct{}

// registerModelNameCallbacks tags each statement's context with its model name
// (e.g. "CustomProfileData") so log lines use the real type, like Rails does,
// instead of guessing from the table name.
func registerModelNameCallbacks(db *gorm.DB) error {
	tag := func(tx *gorm.DB) {
		if s := tx.Statement.Schema; s != nil && tx.Statement.Context != nil {
			tx.Statement.Context = context.WithValue(tx.Statement.Context, modelNameKey{}, s.Name)
		}
	}
	cb := db.Callback()
	for _, err := range []error{
		cb.Create().Before("gorm:create").Register("octopussy:model_name", tag),
		cb.Query().Before("gorm:query").Register("octopussy:model_name", tag),
		cb.Update().Before("gorm:update").Register("octopussy:model_name", tag),
		cb.Delete().Before("gorm:delete").Register("octopussy:model_name", tag),
		cb.Row().Before("gorm:row").Register("octopussy:model_name", tag),
	} {
		if err != nil {
			return fmt.Errorf("register sql log callback: %w", err)
		}
	}
	return nil
}

// modulePath and moduleRoot let appCaller pick the first stack frame that
// belongs to this codebase (skipping gorm, gormigrate and the stdlib), the way
// Rails' backtrace cleaner shows only app frames.
const modulePath = "github.com/thepeacockproject/Octopussy/"

var (
	selfFile   string
	moduleRoot string
)

func init() {
	_, selfFile, _, _ = runtime.Caller(0)
	moduleRoot = strings.TrimSuffix(selfFile, "internal/persistence/sqllog.go")
}

func appCaller() string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		f, more := frames.Next()
		if strings.HasPrefix(f.Function, modulePath) && f.File != selfFile {
			file := strings.TrimPrefix(f.File, moduleRoot)
			file = strings.TrimPrefix(file, modulePath) // -trimpath builds
			return fmt.Sprintf("%s:%d", file, f.Line)
		}
		if !more {
			return ""
		}
	}
}
