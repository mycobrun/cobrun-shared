package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// TestNewLogger tests logger creation with different log levels
func TestNewLogger(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel slog.Level
		addSource     bool
	}{
		{
			name:          "debug level",
			level:         "debug",
			expectedLevel: slog.LevelDebug,
			addSource:     true,
		},
		{
			name:          "info level",
			level:         "info",
			expectedLevel: slog.LevelInfo,
			addSource:     false,
		},
		{
			name:          "warn level",
			level:         "warn",
			expectedLevel: slog.LevelWarn,
			addSource:     false,
		},
		{
			name:          "warning level",
			level:         "warning",
			expectedLevel: slog.LevelWarn,
			addSource:     false,
		},
		{
			name:          "error level",
			level:         "error",
			expectedLevel: slog.LevelError,
			addSource:     false,
		},
		{
			name:          "invalid level defaults to info",
			level:         "invalid",
			expectedLevel: slog.LevelInfo,
			addSource:     false,
		},
		{
			name:          "empty level defaults to info",
			level:         "",
			expectedLevel: slog.LevelInfo,
			addSource:     false,
		},
		{
			name:          "uppercase DEBUG",
			level:         "DEBUG",
			expectedLevel: slog.LevelDebug,
			addSource:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.level)
			if logger == nil {
				t.Fatal("expected logger to be non-nil")
			}
			if logger.level != tt.expectedLevel {
				t.Errorf("expected level %v, got %v", tt.expectedLevel, logger.level)
			}
		})
	}
}

// TestParseLevel tests the parseLevel function
func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestLoggerContext tests context operations
func TestLoggerContext(t *testing.T) {
	logger := NewLogger("info")
	ctx := context.Background()

	// Test WithContext
	newCtx := logger.WithContext(ctx)
	if newCtx == ctx {
		t.Error("expected new context to be different from original")
	}

	// Test FromContext with logger
	retrievedLogger := FromContext(newCtx)
	if retrievedLogger == nil {
		t.Fatal("expected logger from context to be non-nil")
	}

	// Test FromContext without logger (should return default)
	emptyCtx := context.Background()
	defaultLogger := FromContext(emptyCtx)
	if defaultLogger == nil {
		t.Fatal("expected default logger to be non-nil")
	}
	if defaultLogger.level != slog.LevelInfo {
		t.Errorf("expected default logger level to be Info, got %v", defaultLogger.level)
	}
}

// TestLoggerWith tests the With method
func TestLoggerWith(t *testing.T) {
	logger := NewLogger("info")
	newLogger := logger.With("key1", "value1", "key2", 123)

	if newLogger == nil {
		t.Fatal("expected logger with fields to be non-nil")
	}

	// Just verify the logger was created with the correct level preserved
	if newLogger.level != logger.level {
		t.Errorf("expected level to be preserved")
	}
}

// TestLoggerWithService tests WithService helper
func TestLoggerWithService(t *testing.T) {
	logger := NewLogger("info")
	serviceLogger := logger.WithService("test-service")

	if serviceLogger == nil {
		t.Fatal("expected service logger to be non-nil")
	}

	if serviceLogger.level != logger.level {
		t.Errorf("expected level to be preserved")
	}
}

// TestLoggerWithRequestID tests WithRequestID helper
func TestLoggerWithRequestID(t *testing.T) {
	logger := NewLogger("info")
	requestLogger := logger.WithRequestID("req-12345")

	if requestLogger == nil {
		t.Fatal("expected request logger to be non-nil")
	}

	if requestLogger.level != logger.level {
		t.Errorf("expected level to be preserved")
	}
}

// TestLoggerWithUserID tests WithUserID helper
func TestLoggerWithUserID(t *testing.T) {
	logger := NewLogger("info")
	userLogger := logger.WithUserID("user-67890")

	if userLogger == nil {
		t.Fatal("expected user logger to be non-nil")
	}

	if userLogger.level != logger.level {
		t.Errorf("expected level to be preserved")
	}
}

// TestLoggerWithError tests WithError helper
func TestLoggerWithError(t *testing.T) {
	logger := NewLogger("info")
	testErr := errors.New("test error message")
	errorLogger := logger.WithError(testErr)

	if errorLogger == nil {
		t.Fatal("expected error logger to be non-nil")
	}

	if errorLogger.level != logger.level {
		t.Errorf("expected level to be preserved")
	}
}

// TestLogLevels tests different log levels
func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logFunc   func(*Logger)
		shouldLog bool
	}{
		{
			name:     "debug logs when level is debug",
			logLevel: "debug",
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			shouldLog: true,
		},
		{
			name:     "debug does not log when level is info",
			logLevel: "info",
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			shouldLog: false,
		},
		{
			name:     "info logs when level is info",
			logLevel: "info",
			logFunc: func(l *Logger) {
				l.Info("info message")
			},
			shouldLog: true,
		},
		{
			name:     "info does not log when level is warn",
			logLevel: "warn",
			logFunc: func(l *Logger) {
				l.Info("info message")
			},
			shouldLog: false,
		},
		{
			name:     "warn logs when level is warn",
			logLevel: "warn",
			logFunc: func(l *Logger) {
				l.Warn("warn message")
			},
			shouldLog: true,
		},
		{
			name:     "error logs when level is error",
			logLevel: "error",
			logFunc: func(l *Logger) {
				l.Error("error message")
			},
			shouldLog: true,
		},
		{
			name:     "warn does not log when level is error",
			logLevel: "error",
			logFunc: func(l *Logger) {
				l.Warn("warn message")
			},
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := NewLogger(tt.logLevel)
			tt.logFunc(logger)

			w.Close()
			os.Stdout = oldStdout
			io.Copy(&buf, r)

			output := buf.String()
			hasOutput := len(strings.TrimSpace(output)) > 0

			if tt.shouldLog && !hasOutput {
				t.Errorf("expected log output but got none")
			}
			if !tt.shouldLog && hasOutput {
				t.Errorf("expected no log output but got: %s", output)
			}
		})
	}
}

// TestStructuredLogging tests structured logging with fields
func TestStructuredLogging(t *testing.T) {
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewLogger("info")
	logger.Info("test message", "field1", "value1", "field2", 42, "field3", true)

	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)

	output := buf.String()

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	// Verify structured fields
	if logEntry["msg"] != "test message" {
		t.Errorf("expected msg to be 'test message', got: %v", logEntry["msg"])
	}

	if logEntry["field1"] != "value1" {
		t.Errorf("expected field1 to be 'value1', got: %v", logEntry["field1"])
	}

	if logEntry["field2"] != float64(42) {
		t.Errorf("expected field2 to be 42, got: %v", logEntry["field2"])
	}

	if logEntry["field3"] != true {
		t.Errorf("expected field3 to be true, got: %v", logEntry["field3"])
	}
}

// TestJSONOutput tests that logger outputs valid JSON
func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewLogger("info")
	logger.Info("json test message", "key", "value")

	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)

	output := buf.String()

	// Verify it's valid JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v", err)
	}

	// Verify standard fields exist
	if _, ok := logEntry["time"]; !ok {
		t.Error("expected 'time' field in JSON output")
	}

	if _, ok := logEntry["level"]; !ok {
		t.Error("expected 'level' field in JSON output")
	}

	if _, ok := logEntry["msg"]; !ok {
		t.Error("expected 'msg' field in JSON output")
	}
}

// TestLoggerChaining tests chaining multiple With methods
func TestLoggerChaining(t *testing.T) {
	logger := NewLogger("info")
	chainedLogger := logger.
		WithService("test-service").
		WithRequestID("req-123").
		WithUserID("user-456")

	if chainedLogger == nil {
		t.Fatal("expected chained logger to be non-nil")
	}

	// Verify level is preserved through chaining
	if chainedLogger.level != logger.level {
		t.Errorf("expected level to be preserved through chaining")
	}
}

// TestLoggerLevelPreservation tests that log level is preserved across With calls
func TestLoggerLevelPreservation(t *testing.T) {
	logger := NewLogger("error")
	newLogger := logger.With("key", "value")

	if newLogger.level != logger.level {
		t.Errorf("expected level to be preserved, got %v, want %v", newLogger.level, logger.level)
	}
}

// TestMultipleLoggersIndependence tests that multiple loggers are independent
func TestMultipleLoggersIndependence(t *testing.T) {
	logger1 := NewLogger("debug")
	logger2 := NewLogger("error")

	if logger1.level == logger2.level {
		t.Error("expected independent loggers to have different levels")
	}

	// Modifying one logger shouldn't affect the other
	logger1WithField := logger1.With("field1", "value1")
	logger2WithField := logger2.With("field2", "value2")

	if logger1WithField.level != slog.LevelDebug {
		t.Error("logger1 level should remain debug")
	}
	if logger2WithField.level != slog.LevelError {
		t.Error("logger2 level should remain error")
	}
}

// TestContextPropagation tests logger propagation through context
func TestContextPropagation(t *testing.T) {
	logger := NewLogger("info").WithService("test-service")
	ctx := logger.WithContext(context.Background())

	// Simulate passing context through multiple functions
	func1 := func(ctx context.Context) context.Context {
		l := FromContext(ctx)
		l = l.WithRequestID("req-123")
		return l.WithContext(ctx)
	}

	func2 := func(ctx context.Context) *Logger {
		return FromContext(ctx)
	}

	newCtx := func1(ctx)
	finalLogger := func2(newCtx)

	// Verify the final logger is not nil and has correct level
	if finalLogger == nil {
		t.Fatal("expected final logger to be non-nil")
	}

	if finalLogger.level != logger.level {
		t.Errorf("expected level to be preserved through context")
	}
}

// TestDebugSourceInformation tests that source info is added for debug level
func TestDebugSourceInformation(t *testing.T) {
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewLogger("debug")
	logger.Debug("debug with source")

	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)

	output := buf.String()

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	// Check for source field (should exist for debug level)
	if _, ok := logEntry["source"]; !ok {
		t.Error("expected 'source' field in debug log output")
	}
}

// TestInfoNoSourceInformation tests that source info is not added for info level
func TestInfoNoSourceInformation(t *testing.T) {
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewLogger("info")
	logger.Info("info without source")

	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)

	output := buf.String()

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse log output as JSON: %v", err)
	}

	// Check that source field does not exist for info level
	if _, ok := logEntry["source"]; ok {
		t.Error("did not expect 'source' field in info log output")
	}
}
