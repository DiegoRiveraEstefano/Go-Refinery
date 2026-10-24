// Package logger provides a structured, leveled, context-aware logger built purely
// on the Go standard library (io, os, sync, time, encoding/json).
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level represents the severity of a log entry.
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
		return "INFO"
	}
}

// ParseLevel converts a string ("debug", "info", "warn", "error") into a Level.
func ParseLevel(raw string) Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Field represents a typed key-value attribute attached to a structured log record.
type Field struct {
	Key   string
	Value any
}

// F constructs a structured log Field.
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// Err constructs a standard "error" Field from an error value.
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

type ctxKey struct{}

// WithRequestID attaches a correlation request_id to ctx.
func WithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, reqID)
}

// RequestIDFromContext extracts the correlation request_id from ctx, if present.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

// Logger writes thread-safe structured log entries in TEXT (key=value) or JSON format.
type Logger struct {
	mu         sync.Mutex
	out        io.Writer
	minLevel   Level
	jsonFormat bool
	baseFields []Field
}

// New creates a new Logger writing to out. If out is nil, os.Stderr is used.
func New(out io.Writer, minLevel Level, jsonFormat bool) *Logger {
	if out == nil {
		out = os.Stderr
	}
	return &Logger{
		out:        out,
		minLevel:   minLevel,
		jsonFormat: jsonFormat,
	}
}

// Default returns a default INFO-level text Logger writing to os.Stderr.
func Default() *Logger {
	return New(os.Stderr, LevelInfo, false)
}

// Nop returns a Logger that discards all output (useful in unit tests).
func Nop() *Logger {
	return New(io.Discard, LevelError, false)
}

// With returns a child Logger with additional persistent context fields (e.g., component).
func (l *Logger) With(fields ...Field) *Logger {
	if l == nil {
		l = Default()
	}
	combined := make([]Field, 0, len(l.baseFields)+len(fields))
	combined = append(combined, l.baseFields...)
	combined = append(combined, fields...)
	return &Logger{
		out:        l.out,
		minLevel:   l.minLevel,
		jsonFormat: l.jsonFormat,
		baseFields: combined,
	}
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(context.Background(), LevelDebug, msg, fields...)
}

// Info logs a message at INFO level.
func (l *Logger) Info(msg string, fields ...Field) {
	l.log(context.Background(), LevelInfo, msg, fields...)
}

// Warn logs a message at WARN level.
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(context.Background(), LevelWarn, msg, fields...)
}

// Error logs a message at ERROR level.
func (l *Logger) Error(msg string, fields ...Field) {
	l.log(context.Background(), LevelError, msg, fields...)
}

// DebugContext logs a message at DEBUG level including any request_id from ctx.
func (l *Logger) DebugContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelDebug, msg, fields...)
}

// InfoContext logs a message at INFO level including any request_id from ctx.
func (l *Logger) InfoContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelInfo, msg, fields...)
}

// WarnContext logs a message at WARN level including any request_id from ctx.
func (l *Logger) WarnContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelWarn, msg, fields...)
}

// ErrorContext logs a message at ERROR level including any request_id from ctx.
func (l *Logger) ErrorContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelError, msg, fields...)
}

func (l *Logger) log(ctx context.Context, level Level, msg string, fields ...Field) {
	if l == nil {
		return
	}
	if level < l.minLevel {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	reqID := RequestIDFromContext(ctx)

	allFields := make([]Field, 0, len(l.baseFields)+len(fields)+1)
	if reqID != "" {
		allFields = append(allFields, Field{Key: "request_id", Value: reqID})
	}
	allFields = append(allFields, l.baseFields...)
	allFields = append(allFields, fields...)

	var line string
	if l.jsonFormat {
		line = formatJSONLine(now, level, msg, allFields)
	} else {
		line = formatTextLine(now, level, msg, allFields)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = io.WriteString(l.out, line)
}

func formatTextLine(ts string, level Level, msg string, fields []Field) string {
	var sb strings.Builder
	sb.WriteString("ts=")
	sb.WriteString(ts)
	sb.WriteString(" level=")
	sb.WriteString(level.String())
	sb.WriteString(" msg=")
	sb.WriteString(quoteIfNeeded(msg))

	for _, f := range fields {
		if f.Key == "" || f.Value == nil {
			continue
		}
		sb.WriteByte(' ')
		sb.WriteString(f.Key)
		sb.WriteByte('=')
		sb.WriteString(quoteIfNeeded(fmt.Sprintf("%v", f.Value)))
	}
	sb.WriteByte('\n')
	return sb.String()
}

func formatJSONLine(ts string, level Level, msg string, fields []Field) string {
	payload := make(map[string]any, len(fields)+3)
	payload["ts"] = ts
	payload["level"] = level.String()
	payload["msg"] = msg
	for _, f := range fields {
		if f.Key != "" && f.Value != nil {
			payload[f.Key] = f.Value
		}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return formatTextLine(ts, level, msg, fields)
	}
	return string(b) + "\n"
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, " \t\n\r\"=") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
