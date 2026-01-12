package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARNING"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging with rotation and cleanup
type Logger struct {
	mu              sync.RWMutex
	level           LogLevel
	logFile         *os.File
	logger          *log.Logger
	logDir          string
	maxFileSize     int64 // Maximum log file size in bytes (0 = no limit)
	maxAge          int   // Maximum number of days to retain old log files (0 = keep all)
	currentFileSize int64
	rotationEnabled bool
}

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	Level           LogLevel // Minimum log level to output
	OutputMode      string   // "stdout", "file", or "both"
	LogDir          string   // Directory for log files (used when OutputMode is "file" or "both")
	MaxFileSizeMB   int      // Maximum log file size in MB before rotation (0 = no limit)
	MaxAgeDays      int      // Maximum number of days to retain old log files (0 = keep all)
	RotationEnabled bool     // Enable log rotation
}

var (
	globalLogger     *Logger
	globalLoggerOnce sync.Once
)

// InitLogger initializes the global logger with the given configuration
func InitLogger(config LoggerConfig) error {
	var err error
	globalLoggerOnce.Do(func() {
		globalLogger, err = NewLogger(config)
	})
	return err
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		// Fallback to a basic logger if not initialized
		config := LoggerConfig{
			Level:      LogLevelInfo,
			OutputMode: "stdout",
		}
		_ = InitLogger(config)
	}
	return globalLogger
}

// NewLogger creates a new logger instance
func NewLogger(config LoggerConfig) (*Logger, error) {
	logger := &Logger{
		level:           config.Level,
		logDir:          config.LogDir,
		maxFileSize:     int64(config.MaxFileSizeMB) * 1024 * 1024,
		maxAge:          config.MaxAgeDays,
		rotationEnabled: config.RotationEnabled,
	}

	var writers []io.Writer

	// Configure output based on OutputMode
	switch config.OutputMode {
	case "file":
		if config.LogDir == "" {
			return nil, fmt.Errorf("log_dir must be specified when output_mode is 'file'")
		}
		if err := os.MkdirAll(config.LogDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		logFile, err := logger.openLogFile()
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.logFile = logFile
		writers = append(writers, logFile)

	case "both":
		if config.LogDir == "" {
			return nil, fmt.Errorf("log_dir must be specified when output_mode is 'both'")
		}
		if err := os.MkdirAll(config.LogDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		logFile, err := logger.openLogFile()
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.logFile = logFile
		writers = append(writers, logFile, os.Stdout)

	case "stdout":
		fallthrough
	default:
		writers = append(writers, os.Stdout)
	}

	multiWriter := io.MultiWriter(writers...)
	logger.logger = log.New(multiWriter, "", log.LstdFlags)

	// Start cleanup goroutine if rotation is enabled and maxAge is set
	if logger.rotationEnabled && logger.maxAge > 0 && config.OutputMode != "stdout" {
		go logger.cleanupRoutine()
	}

	return logger, nil
}

// openLogFile opens a new log file with the current timestamp
func (l *Logger) openLogFile() (*os.File, error) {
	filename := filepath.Join(l.logDir, fmt.Sprintf("wol-server-%s.log", time.Now().Format("2006-01-02")))

	// Open in append mode, create if doesn't exist
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	l.currentFileSize = info.Size()

	return file, nil
}

// rotate rotates the log file if necessary
func (l *Logger) rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile == nil {
		return nil
	}

	// Close current log file
	if err := l.logFile.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Rename current log file with timestamp
	oldName := l.logFile.Name()
	newName := fmt.Sprintf("%s.%s", oldName, time.Now().Format("20060102-150405"))
	if err := os.Rename(oldName, newName); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Open new log file
	newFile, err := l.openLogFile()
	if err != nil {
		return fmt.Errorf("failed to open new log file: %w", err)
	}

	l.logFile = newFile
	l.currentFileSize = 0

	// Update logger to write to new file
	writers := []io.Writer{newFile}
	if l.logger.Writer() != newFile {
		// If we were writing to both stdout and file, maintain that
		_, ok := l.logger.Writer().(io.Writer)
		if ok {
			writers = append(writers, os.Stdout)
		}
	}
	l.logger.SetOutput(io.MultiWriter(writers...))

	return nil
}

// checkRotation checks if rotation is needed and performs it
func (l *Logger) checkRotation(messageSize int) {
	if !l.rotationEnabled || l.maxFileSize == 0 || l.logFile == nil {
		return
	}

	l.mu.RLock()
	needsRotation := l.currentFileSize+int64(messageSize) > l.maxFileSize
	l.mu.RUnlock()

	if needsRotation {
		if err := l.rotate(); err != nil {
			// Log to stderr if rotation fails
			fmt.Fprintf(os.Stderr, "ERROR: Log rotation failed: %v\n", err)
		}
	}
}

// cleanupRoutine runs periodically to clean up old log files
func (l *Logger) cleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour) // Run once per day
	defer ticker.Stop()

	for range ticker.C {
		if err := l.CleanupOldLogs(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Log cleanup failed: %v\n", err)
		}
	}
}

// CleanupOldLogs removes log files older than maxAge days
func (l *Logger) CleanupOldLogs() error {
	if l.logDir == "" || l.maxAge == 0 {
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -l.maxAge)

	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	removedCount := 0
	var removedSize int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process log files
		if !isLogFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Remove files older than cutoff
		if info.ModTime().Before(cutoffTime) {
			filePath := filepath.Join(l.logDir, entry.Name())
			size := info.Size()
			if err := os.Remove(filePath); err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: Failed to remove old log file %s: %v\n", filePath, err)
			} else {
				removedCount++
				removedSize += size
			}
		}
	}

	if removedCount > 0 {
		l.Info("Log cleanup completed: removed %d files (%.2f MB)", removedCount, float64(removedSize)/(1024*1024))
	}

	return nil
}

// isLogFile checks if a filename is a log file
func isLogFile(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".log" || ext == "" && len(filename) > 4 && filename[len(filename)-4:] == ".log"
}

// log writes a log message at the specified level
func (l *Logger) log(level LogLevel, format string, v ...interface{}) {
	if level < l.level {
		return // Skip messages below the configured level
	}

	message := fmt.Sprintf("[%s] %s", level, fmt.Sprintf(format, v...))

	// Check if rotation is needed before writing
	l.checkRotation(len(message))

	l.mu.Lock()
	l.logger.Print(message)
	if l.logFile != nil {
		l.currentFileSize += int64(len(message) + 1) // +1 for newline
	}
	l.mu.Unlock()
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(LogLevelDebug, format, v...)
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(LogLevelInfo, format, v...)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, v ...interface{}) {
	l.log(LogLevelWarning, format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(LogLevelError, format, v...)
}

// Fatal logs a fatal error message and exits the program
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(LogLevelError, format, v...)
	os.Exit(1)
}

// Close closes the logger and any open log files
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Convenience functions for global logger

// Debug logs a debug message using the global logger
func Debug(format string, v ...interface{}) {
	GetLogger().Debug(format, v...)
}

// Info logs an info message using the global logger
func Info(format string, v ...interface{}) {
	GetLogger().Info(format, v...)
}

// Warning logs a warning message using the global logger
func Warning(format string, v ...interface{}) {
	GetLogger().Warning(format, v...)
}

// Error logs an error message using the global logger
func Error(format string, v ...interface{}) {
	GetLogger().Error(format, v...)
}

// Fatal logs a fatal error message and exits using the global logger
func Fatal(format string, v ...interface{}) {
	GetLogger().Fatal(format, v...)
}
