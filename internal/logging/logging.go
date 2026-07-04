// Package logging provides structured, leveled, file-based logging for ZED.
//
// The TUI owns the terminal, so logs must never go to stdout/stderr. Instead
// they are written to a rotating log file under the user's config directory.
// The logger is safe for concurrent use and supports contextual fields.
package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Level is the severity of a log record.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string to a Level, defaulting to Info.
func ParseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

// Field is a single structured key/value pair attached to a log record.
type Field struct {
	Key   string
	Value any
}

// F is a shorthand constructor for a Field.
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// record is the JSON shape written to disk.
type record struct {
	Time    string         `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"msg"`
	Caller  string         `json:"caller,omitempty"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// Logger writes structured JSON lines to a file with size-based rotation.
type Logger struct {
	mu         sync.Mutex
	file       *os.File
	path       string
	level      Level
	maxBytes   int64
	written    int64
	maxBackups int
	baseFields []Field
	closed     bool
}

// Options configures a Logger.
type Options struct {
	Path       string // full path to the log file
	Level      Level  // minimum level to record
	MaxBytes   int64  // rotate after this many bytes (0 = 10MB default)
	MaxBackups int    // number of rotated files to keep (0 = 3 default)
}

// DefaultPath returns the standard ZED log file location.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "zed", "logs", "zed.log")
}

// New creates a Logger, creating parent directories as needed.
func New(opts Options) (*Logger, error) {
	if opts.Path == "" {
		opts.Path = DefaultPath()
	}
	if opts.MaxBytes == 0 {
		opts.MaxBytes = 10 * 1024 * 1024
	}
	if opts.MaxBackups == 0 {
		opts.MaxBackups = 3
	}
	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	f, err := os.OpenFile(opts.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	info, _ := f.Stat()
	var size int64
	if info != nil {
		size = info.Size()
	}
	return &Logger{
		file:       f,
		path:       opts.Path,
		level:      opts.Level,
		maxBytes:   opts.MaxBytes,
		maxBackups: opts.MaxBackups,
		written:    size,
	}, nil
}

// With returns a child logger that includes the given fields on every record.
func (l *Logger) With(fields ...Field) *Logger {
	child := &Logger{
		file:       l.file,
		path:       l.path,
		level:      l.level,
		maxBytes:   l.maxBytes,
		maxBackups: l.maxBackups,
		baseFields: append(append([]Field{}, l.baseFields...), fields...),
	}
	return child
}

// SetLevel changes the minimum level at runtime.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
}

func (l *Logger) log(level Level, msg string, fields []Field) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed || l.file == nil {
		return
	}

	merged := map[string]any{}
	for _, f := range l.baseFields {
		merged[f.Key] = f.Value
	}
	for _, f := range fields {
		merged[f.Key] = f.Value
	}

	rec := record{
		Time:    time.Now().Format(time.RFC3339Nano),
		Level:   level.String(),
		Message: msg,
		Caller:  caller(3),
	}
	if len(merged) > 0 {
		rec.Fields = merged
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	data = append(data, '\n')

	if l.written+int64(len(data)) > l.maxBytes {
		l.rotateLocked()
	}
	n, _ := l.file.Write(data)
	l.written += int64(n)
}

// rotateLocked renames the current file and opens a fresh one. Caller holds mu.
func (l *Logger) rotateLocked() {
	_ = l.file.Close()
	// shift backups: zed.log.2 -> zed.log.3, etc.
	for i := l.maxBackups - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", l.path, i)
		dst := fmt.Sprintf("%s.%d", l.path, i+1)
		_ = os.Rename(src, dst)
	}
	_ = os.Rename(l.path, l.path+".1")

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	l.file = f
	l.written = 0
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, fields ...Field) { l.log(LevelDebug, msg, fields) }

// Info logs at info level.
func (l *Logger) Info(msg string, fields ...Field) { l.log(LevelInfo, msg, fields) }

// Warn logs at warn level.
func (l *Logger) Warn(msg string, fields ...Field) { l.log(LevelWarn, msg, fields) }

// Error logs at error level.
func (l *Logger) Error(msg string, fields ...Field) { l.log(LevelError, msg, fields) }

// Close flushes and closes the underlying file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed || l.file == nil {
		return nil
	}
	l.closed = true
	return l.file.Close()
}

// caller returns "file:line" for the given stack skip depth.
func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// Nop returns a logger that discards everything. Useful in tests.
func Nop() *Logger {
	return &Logger{level: LevelError + 1, closed: true}
}

// sortedKeys is a small helper for deterministic field ordering in tests.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var _ = sortedKeys // reserved for future formatted output
