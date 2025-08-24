package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
}

// Config represents logger configuration
type Config struct {
	Level      string `json:"level"`
	Format     string `json:"format"` // "json" or "text"
	Output     string `json:"output"` // "stdout", "stderr", or file path
	TimeFormat string `json:"time_format"`
}

// NewLogger creates a new logger instance
func NewLogger(config Config) (*Logger, error) {
	logger := logrus.New()
	
	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logger.SetLevel(level)
	
	// Set formatter
	switch config.Format {
	case "json":
		formatter := &logrus.JSONFormatter{}
		if config.TimeFormat != "" {
			formatter.TimestampFormat = config.TimeFormat
		} else {
			formatter.TimestampFormat = time.RFC3339
		}
		logger.SetFormatter(formatter)
	case "text", "":
		formatter := &logrus.TextFormatter{
			FullTimestamp: true,
		}
		if config.TimeFormat != "" {
			formatter.TimestampFormat = config.TimeFormat
		} else {
			formatter.TimestampFormat = time.RFC3339
		}
		logger.SetFormatter(formatter)
	default:
		return nil, fmt.Errorf("invalid log format: %s", config.Format)
	}
	
	// Set output
	switch config.Output {
	case "stdout", "":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		// Assume it's a file path
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.SetOutput(file)
	}
	
	return &Logger{Logger: logger}, nil
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() *Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logger.SetOutput(os.Stdout)
	
	return &Logger{Logger: logger}
}

// WithField creates a new logger entry with a single field
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields creates a new logger entry with multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields(fields))
}

// WithError creates a new logger entry with an error field
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// Debug logs a debug message
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(args ...interface{}) {
	l.Logger.Warn(args...)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warnf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatalf(format, args...)
}

// Panic logs a panic message and panics
func (l *Logger) Panic(args ...interface{}) {
	l.Logger.Panic(args...)
}

// Panicf logs a formatted panic message and panics
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.Logger.Panicf(format, args...)
}

// SetLevel sets the logger level
func (l *Logger) SetLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	l.Logger.SetLevel(logLevel)
	return nil
}

// GetLevel returns the current logger level
func (l *Logger) GetLevel() string {
	return l.Logger.GetLevel().String()
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.Logger.IsLevelEnabled(logrus.DebugLevel)
}

// IsInfoEnabled returns true if info logging is enabled
func (l *Logger) IsInfoEnabled() bool {
	return l.Logger.IsLevelEnabled(logrus.InfoLevel)
}

// Clone creates a copy of the logger
func (l *Logger) Clone() *Logger {
	newLogger := logrus.New()
	newLogger.SetLevel(l.Logger.GetLevel())
	newLogger.SetFormatter(l.Logger.Formatter)
	newLogger.SetOutput(l.Logger.Out)
	
	return &Logger{Logger: newLogger}
}