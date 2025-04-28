package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	// LogLevelDebug is used for debug messages
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is used for informational messages
	LogLevelInfo
	// LogLevelWarning is used for warning messages
	LogLevelWarning
	// LogLevelError is used for error messages
	LogLevelError
)

// Logger represents a simple logging system
type Logger struct {
	file     *os.File
	logLevel LogLevel
}

// NewLogger creates a new logger that writes to the specified file path
func NewLogger(logFilePath string, level LogLevel) (*Logger, error) {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file with append mode
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &Logger{
		file:     file,
		logLevel: level,
	}, nil
}

// SetLogLevel sets the current log level
func (l *Logger) SetLogLevel(level LogLevel) {
	l.logLevel = level
}

// Close closes the logger
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// logWithLevel logs a message with the specified level
func (l *Logger) logWithLevel(level LogLevel, message string) {
	if level < l.logLevel {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	var levelStr string

	switch level {
	case LogLevelDebug:
		levelStr = "DEBUG"
	case LogLevelInfo:
		levelStr = "INFO"
	case LogLevelWarning:
		levelStr = "WARNING"
	case LogLevelError:
		levelStr = "ERROR"
	default:
		levelStr = "UNKNOWN"
	}

	logMessage := fmt.Sprintf("[%s] [%s] %s\n", timestamp, levelStr, message)
	l.file.WriteString(logMessage)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logWithLevel(LogLevelDebug, message)
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logWithLevel(LogLevelInfo, message)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logWithLevel(LogLevelWarning, message)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logWithLevel(LogLevelError, message)
}

// MultiWriter creates a writer that duplicates its writes to all the provided writers
func MultiWriter(writers ...io.Writer) io.Writer {
	return io.MultiWriter(writers...)
}

// CreateLogFile creates a log file with a timestamp in the filename
func CreateLogFile(logDir, prefix string) (*os.File, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(logDir, fmt.Sprintf("%s_%s.log", prefix, timestamp))

	// Create and open the file
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return file, nil
}
