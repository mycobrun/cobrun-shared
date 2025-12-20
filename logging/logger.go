// Package logging provides structured logging with Azure Application Insights integration.
package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// contextKey is used for storing logger in context.
type contextKey struct{}

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	level slog.Level
}

// NewLogger creates a new structured logger.
func NewLogger(level string) *Logger {
	l := parseLevel(level)

	opts := &slog.HandlerOptions{
		Level:     l,
		AddSource: l == slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		level:  l,
	}
}

// WithContext returns a new context with the logger.
func (l *Logger) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext retrieves the logger from context.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(contextKey{}).(*Logger); ok {
		return l
	}
	return NewLogger("info")
}

// With returns a new logger with additional attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
		level:  l.level,
	}
}

// WithService returns a logger with service name.
func (l *Logger) WithService(name string) *Logger {
	return l.With("service", name)
}

// WithRequestID returns a logger with request ID.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return l.With("request_id", requestID)
}

// WithUserID returns a logger with user ID.
func (l *Logger) WithUserID(userID string) *Logger {
	return l.With("user_id", userID)
}

// WithError returns a logger with error.
func (l *Logger) WithError(err error) *Logger {
	return l.With("error", err.Error())
}

// Helper to parse log level string.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Standard logging functions with context support.

// Debug logs at debug level.
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs at info level.
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs at error level.
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Fatal logs at error level and exits.
func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	os.Exit(1)
}
