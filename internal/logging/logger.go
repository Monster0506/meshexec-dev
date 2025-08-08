package logging

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger provides structured logging functionality
type Logger struct {
	logger zerolog.Logger
}

// NewLogger creates a new logger with the specified level
func NewLogger(level string) *Logger {
	// Parse log level
	logLevel := parseLogLevel(level)

	// Configure zerolog
	zerolog.SetGlobalLevel(logLevel)

	// Always use pretty console logging for better readability
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	return &Logger{
		logger: log.Logger,
	}
}

// parseLogLevel converts a string level to zerolog level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "none", "off", "silent":
		return zerolog.Disabled
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Debug().Fields(fields).Msg(msg)
	} else {
		l.logger.Debug().Msg(msg)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Info().Fields(fields).Msg(msg)
	} else {
		l.logger.Info().Msg(msg)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Warn().Fields(fields).Msg(msg)
	} else {
		l.logger.Warn().Msg(msg)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, err error, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Error().Err(err).Fields(fields).Msg(msg)
	} else {
		l.logger.Error().Err(err).Msg(msg)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error, fields map[string]interface{}) {
	if fields != nil {
		l.logger.Fatal().Err(err).Fields(fields).Msg(msg)
	} else {
		l.logger.Fatal().Err(err).Msg(msg)
	}
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		logger: l.logger.With().Interface(key, value).Logger(),
	}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{
		logger: l.logger.With().Fields(fields).Logger(),
	}
}

// GetLogger returns the underlying zerolog logger
func (l *Logger) GetLogger() zerolog.Logger {
	return l.logger
}
