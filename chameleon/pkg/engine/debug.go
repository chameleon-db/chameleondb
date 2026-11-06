package engine

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// DebugLevel defines verbosity
type DebugLevel int

const (
	DebugNone DebugLevel = iota
	DebugSQL
	DebugTrace
	DebugExplain
)

// DebugContext holds debug configuration
type DebugContext struct {
	Level  DebugLevel
	Writer io.Writer // Where to write (stdout, file, etc)

	// Future expansion
	EnableTiming    bool
	EnableProfiling bool
	ColorOutput     bool
}

var sensitiveDebugFragments = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"private_key", "credential", "hash", "salt",
}

// DefaultDebugContext for production
func DefaultDebugContext() *DebugContext {
	return &DebugContext{
		Level:       DebugNone,
		Writer:      os.Stdout,
		ColorOutput: true,
	}
}

// DebugContextFromEnv reads CHAMELEON_DEBUG from the environment:
// "1", "true" or "sql" log SQL; "trace" adds timings; "explain" is the most
// verbose level. Anything else leaves debugging off.
func DebugContextFromEnv() *DebugContext {
	level := DebugNone

	switch strings.ToLower(strings.TrimSpace(os.Getenv("CHAMELEON_DEBUG"))) {
	case "1", "true", "sql":
		level = DebugSQL
	case "trace":
		level = DebugTrace
	case "explain":
		level = DebugExplain
	}

	return &DebugContext{
		Level:       level,
		Writer:      os.Stdout,
		ColorOutput: true,
	}
}

// Log writes debug output
func (dc *DebugContext) Log(level DebugLevel, format string, args ...interface{}) {
	if dc.Level < level {
		return
	}

	var prefix string
	if dc.ColorOutput {
		prefix = colorPrefix(level)
	} else {
		prefix = textPrefix(level)
	}

	fmt.Fprintf(dc.Writer, prefix+format+"\n", args...)
}

// LogSQL logs generated SQL
func (dc *DebugContext) LogSQL(sql string) {
	if dc.Level < DebugSQL {
		return
	}

	if dc.ColorOutput {
		fmt.Fprintf(dc.Writer, "\n\033[36m[SQL]\033[0m\n%s\n\n", sql)
	} else {
		fmt.Fprintf(dc.Writer, "\n[SQL]\n%s\n\n", sql)
	}
}

// LogMutationSQL logs generated mutation SQL using the configured writer.
func (dc *DebugContext) LogMutationSQL(operation, entity, sql string) {
	if dc.Level < DebugSQL {
		return
	}

	if dc.ColorOutput {
		fmt.Fprintf(dc.Writer, "\n\033[36m[SQL]\033[0m %s %s\n%s\n", operation, entity, sql)
	} else {
		fmt.Fprintf(dc.Writer, "\n[SQL] %s %s\n%s\n", operation, entity, sql)
	}
}

// LogMutationTrace logs the duration and row count of a mutation.
func (dc *DebugContext) LogMutationTrace(operation, entity string, duration time.Duration, rowCount int) {
	dc.Log(DebugTrace, "%s on %s: %v, %d rows", operation, entity, duration, rowCount)
}

// LogNamedParams logs parameters with field names so sensitive values can be
// masked. Values without a matching field name are masked conservatively.
func (dc *DebugContext) LogNamedParams(labels []string, values []interface{}) {
	if dc.Level < DebugSQL || len(values) == 0 {
		return
	}

	if dc.ColorOutput {
		fmt.Fprintf(dc.Writer, "\033[36m[PARAMS]\033[0m\n")
	} else {
		fmt.Fprintf(dc.Writer, "[PARAMS]\n")
	}
	aligned := len(labels) == len(values)
	for i, value := range values {
		display := "[REDACTED]"
		if aligned && !isSensitiveDebugField(labels[i]) {
			display = fmt.Sprintf("%v", value)
		}
		fmt.Fprintf(dc.Writer, "$%d = %s\n", i+1, display)
	}
	fmt.Fprintf(dc.Writer, "\n")
}

// LogEagerSQL logs the SQL of an eager-loading query for a relation. The
// parent ids are bound at execution time, so only their placeholder is shown.
func (dc *DebugContext) LogEagerSQL(relation, sql string, params []SQLParam) {
	if dc.Level < DebugSQL {
		return
	}

	if dc.ColorOutput {
		fmt.Fprintf(dc.Writer, "\033[36m[SQL]\033[0m include %s\n%s\n\n", relation, sql)
	} else {
		fmt.Fprintf(dc.Writer, "[SQL] include %s\n%s\n\n", relation, sql)
	}
	dc.LogParams(params)
}

// LogEagerTrace logs the duration and row count of an eager-loading query.
func (dc *DebugContext) LogEagerTrace(relation string, duration time.Duration, rowCount int) {
	dc.Log(DebugTrace, "include %s: %v, %d rows", relation, duration, rowCount)
}

// LogParams logs the parameters of a statement, separately from its SQL text.
func (dc *DebugContext) LogParams(params []SQLParam) {
	if dc.Level < DebugSQL || len(params) == 0 {
		return
	}

	if dc.ColorOutput {
		fmt.Fprintf(dc.Writer, "\033[36m[PARAMS]\033[0m\n")
	} else {
		fmt.Fprintf(dc.Writer, "[PARAMS]\n")
	}
	for i := range params {
		fmt.Fprintf(dc.Writer, "$%d = [REDACTED]\n", i+1)
	}
	fmt.Fprintf(dc.Writer, "\n")
}

func isSensitiveDebugField(name string) bool {
	lower := strings.ToLower(name)
	for _, fragment := range sensitiveDebugFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

// LogQuery logs full query trace
func (dc *DebugContext) LogQuery(sql string, duration time.Duration, rowCount int) {
	if dc.Level < DebugTrace {
		return
	}

	fmt.Fprintf(dc.Writer, "\n")
	fmt.Fprintf(dc.Writer, "┌─────────────────────────────────────\n")
	fmt.Fprintf(dc.Writer, "│ Query Trace\n")
	fmt.Fprintf(dc.Writer, "├─────────────────────────────────────\n")
	fmt.Fprintf(dc.Writer, "│ SQL:\n│   %s\n", sql)
	fmt.Fprintf(dc.Writer, "│ Duration: %v\n", duration)
	fmt.Fprintf(dc.Writer, "│ Rows: %d\n", rowCount)
	fmt.Fprintf(dc.Writer, "└─────────────────────────────────────\n\n")
}

func colorPrefix(level DebugLevel) string {
	switch level {
	case DebugSQL:
		return "\033[36m[DEBUG]\033[0m "
	case DebugTrace:
		return "\033[33m[TRACE]\033[0m "
	case DebugExplain:
		return "\033[35m[EXPLAIN]\033[0m "
	default:
		return ""
	}
}

func textPrefix(level DebugLevel) string {
	switch level {
	case DebugSQL:
		return "[DEBUG] "
	case DebugTrace:
		return "[TRACE] "
	case DebugExplain:
		return "[EXPLAIN] "
	default:
		return ""
	}
}
