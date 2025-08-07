package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zerolog.Level
	}{
		{"debug level", "debug", zerolog.DebugLevel},
		{"info level", "info", zerolog.InfoLevel},
		{"warn level", "warn", zerolog.WarnLevel},
		{"warning level", "warning", zerolog.WarnLevel},
		{"error level", "error", zerolog.ErrorLevel},
		{"fatal level", "fatal", zerolog.FatalLevel},
		{"panic level", "panic", zerolog.PanicLevel},
		{"unknown level", "unknown", zerolog.InfoLevel},
		{"empty level", "", zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.level)
			assert.NotNil(t, logger)
			assert.Equal(t, tt.expected, zerolog.GlobalLevel())
		})
	}
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	
	// Create logger with custom output for testing
	logger := &Logger{
		logger: zerolog.New(&buf).With().Timestamp().Logger(),
	}

	// Test basic info message
	logger.Info("test message", nil)
	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "info")

	// Test info message with fields
	buf.Reset()
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}
	logger.Info("test message with fields", fields)
	output = buf.String()
	assert.Contains(t, output, "test message with fields")
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "key2")
	assert.Contains(t, output, "123")
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		logger: zerolog.New(&buf).With().Timestamp().Logger(),
	}

	// Test error message
	testErr := assert.AnError
	logger.Error("test error", testErr, nil)
	output := buf.String()
	assert.Contains(t, output, "test error")
	assert.Contains(t, output, "error")
	assert.Contains(t, output, testErr.Error())

	// Test error message with fields
	buf.Reset()
	fields := map[string]interface{}{
		"component": "test",
		"operation": "logging",
	}
	logger.Error("test error with fields", testErr, fields)
	output = buf.String()
	assert.Contains(t, output, "test error with fields")
	assert.Contains(t, output, "component")
	assert.Contains(t, output, "test")
	assert.Contains(t, output, "operation")
	assert.Contains(t, output, "logging")
}

func TestLogger_WithField(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		logger: zerolog.New(&buf).With().Timestamp().Logger(),
	}

	// Test with single field
	loggerWithField := logger.WithField("component", "test")
	loggerWithField.Info("test message", nil)
	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "component")
	assert.Contains(t, output, "test")
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		logger: zerolog.New(&buf).With().Timestamp().Logger(),
	}

	// Test with multiple fields
	fields := map[string]interface{}{
		"component": "test",
		"version":   "1.0.0",
	}
	loggerWithFields := logger.WithFields(fields)
	loggerWithFields.Info("test message", nil)
	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "component")
	assert.Contains(t, output, "test")
	assert.Contains(t, output, "version")
	assert.Contains(t, output, "1.0.0")
}

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	
	// Create logger with debug level enabled
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	logger := &Logger{
		logger: zerolog.New(&buf).With().Timestamp().Logger(),
	}

	// Test debug level
	buf.Reset()
	logger.Debug("debug message", nil)
	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "debug")

	// Test warn level
	buf.Reset()
	logger.Warn("warn message", nil)
	output = buf.String()
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "warn")
}

func TestLogger_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		logger: zerolog.New(&buf).With().Timestamp().Logger(),
	}

	// Test JSON output format
	logger.Info("test message", map[string]interface{}{
		"key": "value",
		"num": 42,
	})

	output := buf.String()
	// Should be valid JSON
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(output), &jsonData)
	require.NoError(t, err)

	// Check for expected fields
	assert.Equal(t, "test message", jsonData["message"])
	assert.Equal(t, "info", jsonData["level"])
	assert.Equal(t, "value", jsonData["key"])
	assert.Equal(t, float64(42), jsonData["num"]) // JSON numbers are float64
}

func TestLogger_ConsoleOutput(t *testing.T) {
	var buf bytes.Buffer
	
	// Create logger with console output (like debug mode)
	logger := &Logger{
		logger: zerolog.New(zerolog.ConsoleWriter{Out: &buf}).With().Timestamp().Logger(),
	}

	logger.Info("test message", map[string]interface{}{
		"key": "value",
	})

	output := buf.String()
	// Console output should be human-readable, not JSON
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
	assert.False(t, strings.HasPrefix(strings.TrimSpace(output), "{"))
}
