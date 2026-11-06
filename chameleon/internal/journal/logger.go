package journal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Entry represents a single journal entry
type Entry struct {
	Timestamp time.Time              `json:"timestamp"`
	Action    string                 `json:"action"`
	Status    string                 `json:"status"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  int64                  `json:"duration_ms,omitempty"`
}

// Logger is an append-only journal logger
type Logger struct {
	journalDir string
	mu         sync.Mutex
	indexMu    sync.Mutex
}

const (
	keyStatus   = "status"
	keyError    = "error"
	keyDuration = "duration_ms"
	detailMark  = "detail."
)

var errorStatuses = map[string]bool{
	"error":            true,
	"failed":           true,
	"denied":           true,
	"aborted_readonly": true,
}

var dailyLogName = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.log$`)

// NewLogger creates a new journal logger
func NewLogger(journalDir string) (*Logger, error) {
	// Create directory if not exists
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create journal directory: %w", err)
	}

	return &Logger{
		journalDir: journalDir,
	}, nil
}

// Log appends an entry to the journal
func (l *Logger) Log(action, status string, details map[string]interface{}, err error) error {
	entry := Entry{
		Timestamp: time.Now(),
		Action:    action,
		Status:    status,
		Details:   details,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	return l.logEntry(&entry)
}

// LogMigration logs a migration event
func (l *Logger) LogMigration(version string, status string, duration int64, backupPath string, details map[string]interface{}) error {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["version"] = version
	details["backup_path"] = backupPath

	entry := Entry{
		Timestamp: time.Now(),
		Action:    "migrate",
		Status:    status,
		Details:   details,
		Duration:  duration,
	}

	return l.logEntry(&entry)
}

// LogSchema logs a schema event
func (l *Logger) LogSchema(action string, status string, details map[string]interface{}) error {
	return l.Log(action, status, details, nil)
}

// LogError logs an error event
func (l *Logger) LogError(action string, err error, details map[string]interface{}) error {
	return l.Log(action, "error", details, err)
}

// logEntry writes entry to file and updates index
func (l *Logger) logEntry(entry *Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	logFile := l.getLogFile()

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	line := l.formatEntry(entry)
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("failed to write to log: %w", err)
	}

	// Update index
	if err := l.updateIndex(entry); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to update index: %v\n", err)
	}

	return nil
}

// formatEntry formats an entry as a single Unix-style log line.
//
// Format: 2026-02-12T10:15:00Z [ACTION] status=ok key1=val1 key2="val with spaces" duration_ms=5 error="msg"
//
// Detail keys are sorted so the output is deterministic, and every value that
// contains whitespace, quotes or control characters is quoted, so one entry is
// always exactly one line that parseEntry can read back.
func (l *Logger) formatEntry(e *Entry) string {
	var b strings.Builder

	b.WriteString(e.Timestamp.Format(time.RFC3339))
	b.WriteString(" [")
	b.WriteString(sanitizeAction(e.Action))
	b.WriteString("] ")
	b.WriteString(keyStatus + "=")
	b.WriteString(quoteValue(e.Status))

	keys := make([]string, 0, len(e.Details))
	for k := range e.Details {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		b.WriteByte(' ')
		b.WriteString(detailKey(k))
		b.WriteByte('=')
		b.WriteString(quoteValue(fmt.Sprint(e.Details[k])))
	}

	if e.Duration > 0 {
		b.WriteString(fmt.Sprintf(" %s=%d", keyDuration, e.Duration))
	}

	if e.Error != "" {
		b.WriteString(" " + keyError + "=" + strconv.Quote(e.Error))
	}

	return b.String()
}

func sanitizeAction(action string) string {
	return strings.Map(func(r rune) rune {
		if r == '[' || r == ']' || r <= ' ' || r == 0x7f {
			return '_'
		}
		return r
	}, action)
}

func detailKey(k string) string {
	k = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-', r == '.':
			return r
		}
		return '_'
	}, k)

	if k == "" {
		k = "_"
	}

	switch k {
	case keyStatus, keyError, keyDuration:
		return detailMark + k
	}

	return k
}

func quoteValue(s string) string {
	quoted := strconv.Quote(s)
	if s != "" && quoted == `"`+s+`"` && !strings.ContainsAny(s, " =") {
		return s
	}
	return quoted
}

// getLogFile returns the path to today's log file
func (l *Logger) getLogFile() string {
	today := time.Now().Format("2006-01-02")
	return filepath.Join(l.journalDir, today+".log")
}

// updateIndex updates the daily index
func (l *Logger) updateIndex(e *Entry) error {
	l.indexMu.Lock()
	defer l.indexMu.Unlock()

	indexFile := filepath.Join(l.journalDir, "index.json")

	// Load existing index
	var index map[string]interface{}
	if data, err := os.ReadFile(indexFile); err == nil {
		if err := json.Unmarshal(data, &index); err != nil {
			return err
		}
	} else {
		index = make(map[string]interface{})
	}

	// Update metadata
	today := time.Now().Format("2006-01-02")
	if index["date"] != today {
		index["date"] = today
		index["entries"] = 0
		index["by_action"] = make(map[string]interface{})
	}

	// Increment counters - handle float64 from JSON
	var entries int
	if val, exists := index["entries"]; exists {
		if floatVal, ok := val.(float64); ok {
			entries = int(floatVal)
		} else if intVal, ok := val.(int); ok {
			entries = intVal
		}
	}
	index["entries"] = entries + 1

	// Update by_action map
	byAction, ok := index["by_action"].(map[string]interface{})
	if !ok {
		byAction = make(map[string]interface{})
		index["by_action"] = byAction
	}

	count := byAction[e.Action]
	if count == nil {
		byAction[e.Action] = 1
	} else {
		// Handle both int and float64 from JSON
		var countVal int
		if floatVal, ok := count.(float64); ok {
			countVal = int(floatVal)
		} else if intVal, ok := count.(int); ok {
			countVal = intVal
		}
		byAction[e.Action] = countVal + 1
	}

	// Save index
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexFile, data, 0644)
}

func (l *Logger) logFiles() ([]string, error) {
	dirEntries, err := os.ReadDir(l.journalDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, de := range dirEntries {
		if !de.IsDir() && dailyLogName.MatchString(de.Name()) {
			files = append(files, filepath.Join(l.journalDir, de.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

func (l *Logger) readFile(path string) ([]*Entry, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var entries []*Entry
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		entry, err := l.parseEntry(line)
		if err != nil {
			// Skip unparseable lines
			continue
		}
		entries = append(entries, entry)
		lines = append(lines, line)
	}

	return entries, lines, nil
}

func (l *Logger) collect(match func(entry *Entry, line string) bool) ([]*Entry, error) {
	files, err := l.logFiles()
	if err != nil {
		return nil, err
	}
	result := []*Entry{}
	for _, file := range files {
		entries, lines, err := l.readFile(file)
		if err != nil {
			return nil, err
		}
		for i, entry := range entries {
			if match(entry, lines[i]) {
				result = append(result, entry)
			}
		}
	}

	return result, nil
}

// Last returns the last N entries, reading back across days when the
// current day has fewer than N.
func (l *Logger) Last(n int) ([]*Entry, error) {
	if n <= 0 {
		return []*Entry{}, nil
	}
	files, err := l.logFiles()
	if err != nil {
		return nil, err
	}
	var collected []*Entry
	for i := len(files) - 1; i >= 0 && len(collected) < n; i-- {
		entries, _, err := l.readFile(files[i])
		if err != nil {
			return nil, err
		}
		collected = append(append([]*Entry{}, entries...), collected...)
	}
	if len(collected) > n {
		collected = collected[len(collected)-n:]
	}
	if collected == nil {
		collected = []*Entry{}
	}
	return collected, nil
}

// Errors returns every failed operation.
func (l *Logger) Errors() ([]*Entry, error) {
	return l.collect(func(e *Entry, _ string) bool {
		return e.Error != "" || errorStatuses[e.Status]
	})
}

// Migrations returns all migration entries
func (l *Logger) Migrations() ([]*Entry, error) {
	return l.collect(func(e *Entry, _ string) bool {
		return e.Action == "migrate"
	})
}

func (l *Logger) Search(query string, limit int) ([]*Entry, error) {
	needle := strings.ToLower(query)
	entries, err := l.collect(func(_ *Entry, line string) bool {
		return strings.Contains(strings.ToLower(line), needle)
	})
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, nil
}

func (l *Logger) parseEntry(line string) (*Entry, error) {
	return parseEntry(line)
}

func parseEntry(line string) (*Entry, error) {
	line = strings.TrimSpace(line)
	sep := strings.IndexByte(line, ' ')
	if sep < 0 {
		return nil, fmt.Errorf("invalid log line format")
	}
	timestamp, err := time.Parse(time.RFC3339, line[:sep])
	if err != nil {
		return nil, err
	}
	rest := strings.TrimLeft(line[sep:], " ")
	if !strings.HasPrefix(rest, "[") {
		return nil, fmt.Errorf("invalid log line format")
	}
	end := strings.IndexByte(rest, ']')
	if end < 0 {
		return nil, fmt.Errorf("invalid log line format")
	}
	entry := &Entry{
		Timestamp: timestamp,
		Action:    rest[1:end],
	}

	for _, kv := range parseFields(rest[end+1:]) {
		key, value := kv[0], kv[1]
		if strings.HasPrefix(key, detailMark) {
			if name := strings.TrimPrefix(key, detailMark); name == keyStatus || name == keyError || name == keyDuration {
				key = "\x00" + name
			}
		}
		switch key {
		case keyStatus:
			entry.Status = value
		case keyError:
			entry.Error = value
		case keyDuration:
			if ms, err := strconv.ParseInt(value, 10, 64); err == nil {
				entry.Duration = ms
			}
		default:
			if entry.Details == nil {
				entry.Details = make(map[string]interface{})
			}
			entry.Details[strings.TrimPrefix(key, "\x00")] = value
		}
	}
	return entry, nil
}

func parseFields(s string) [][2]string {
	var fields [][2]string
	for {
		s = strings.TrimLeft(s, " ")
		if s == "" {
			return fields
		}
		eq := strings.IndexAny(s, "= ")
		if eq < 0 || s[eq] == ' ' {
			if eq < 0 {
				return fields
			}
			s = s[eq:]
			continue
		}
		key := s[:eq]
		s = s[eq+1:]
		if strings.HasPrefix(s, `"`) {
			if quoted, err := strconv.QuotedPrefix(s); err == nil {
				if value, err := strconv.Unquote(quoted); err == nil {
					fields = append(fields, [2]string{key, value})
					s = s[len(quoted):]
					continue
				}
			}
		}
		end := strings.IndexByte(s, ' ')
		if end < 0 {
			end = len(s)
		}
		fields = append(fields, [2]string{key, s[:end]})
		s = s[end:]
	}
}
