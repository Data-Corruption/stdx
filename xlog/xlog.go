// Package xlog extends log with a leveled, concurrent-safe logger. Logs are
// written to disk using a buffered rotating writer in a sub-package "rlog",
// and can be filtered by level: debug, info, warn, error, or none.
//
// The logger prefixes messages with the process ID and supports
// dynamic log level changes, log formatting customization, and safe
// shutdown via Close().
//
// Usage:
//
//	package main
//
//	import "github.com/Data-Corruption/stdx/xlog"
//
//	// Create a logger
//	logDir := "./logs"
//	logger, err := xlog.New(logDir, "debug")
//	if err != nil {
//		log.Fatalf("Failed to create logger: %v", err)
//	}
//	defer logger.Close() // Ensure logs are flushed
//
//	// Log using methods
//	logger.Info("Application started")
//	logger.Debugf("Configuration value: %s", "some_value")
//
//	// Log using context
//	ctx := context.Background()
//	ctx = xlog.IntoContext(ctx, logger) // Place logger into context
//	xlog.Info(ctx, "Hello")             // Uses logger placed in context
//	xlog.Warn(ctx, "Warning message")
package xlog

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Data-Corruption/stdx/xlog/rlog"
)

const (
	levelDebug int = iota
	levelInfo
	levelWarn
	levelError
	levelNone
)

var (
	ErrInvalidLogLevel = fmt.Errorf("invalid log level")
	ErrClosed          = fmt.Errorf("logger closed")
)

type Logger struct {
	closeMu sync.Mutex
	closed  atomic.Uint32
	level   atomic.Uint32
	writer  *rlog.Writer
	// levels use std lib log package for formatting, flags, etc.
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
}

type ctxKey struct{}

func IntoContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return logger
	}
	return nil
}

// New creates a new logger instance with the given directory path and log level.
// Levels are: debug, info, warn, error, none (case-insensitive).
func New(dirPath string, level string) (*Logger, error) {
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create log directory '%s': %w", dirPath, err)
	}
	var writer *rlog.Writer
	var err error
	if writer, err = rlog.NewWriter(rlog.Config{DirPath: dirPath}); err != nil {
		return nil, fmt.Errorf("failed to initialize rlog writer in directory '%s': %w", dirPath, err)
	}
	pid := os.Getpid()
	l := &Logger{
		writer: writer,
		debug:  log.New(io.Discard, fmt.Sprintf("[PID:%d]DEBUG: ", pid), log.Ldate|log.Ltime|log.Llongfile),
		info:   log.New(io.Discard, fmt.Sprintf("[PID:%d]INFO: ", pid), log.LstdFlags),
		warn:   log.New(io.Discard, fmt.Sprintf("[PID:%d]WARN: ", pid), log.LstdFlags),
		error:  log.New(io.Discard, fmt.Sprintf("[PID:%d]ERROR: ", pid), log.LstdFlags),
	}
	l.closed.Store(0)
	l.level.Store(uint32(levelNone))
	return l, l.SetLevel(level)
}

func (l *Logger) isLevelEnabled(level int) bool {
	if l.IsClosed() {
		return false
	}
	return l.level.Load() <= uint32(level)
}

// Writer returns a pointer to the underlying rlog.Writer.
// Useful for injecting other data into the log stream, manual flushes,
// or checking health via Logger.Writer().Error()
func (l *Logger) Writer() *rlog.Writer {
	l.closeMu.Lock()
	defer l.closeMu.Unlock()
	if l.IsClosed() {
		return nil
	}
	return l.writer
}

func (l *Logger) Debug(v ...interface{}) {
	if l.isLevelEnabled(levelDebug) {
		if err := l.debug.Output(2, fmt.Sprint(v...)); err != nil {
			log.Printf("logger: failed to write debug log entry: %v", err)
		}
	}
}

func Debug(ctx context.Context, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelDebug) {
			if err := l.debug.Output(2, fmt.Sprint(v...)); err != nil {
				log.Printf("logger: failed to write debug log entry: %v", err)
			}
		}
	}
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.isLevelEnabled(levelDebug) {
		if err := l.debug.Output(2, fmt.Sprintf(format, v...)); err != nil {
			log.Printf("logger: failed to write debugf log entry: %v", err)
		}
	}
}

func Debugf(ctx context.Context, format string, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelDebug) {
			if err := l.debug.Output(2, fmt.Sprintf(format, v...)); err != nil {
				log.Printf("logger: failed to write debugf log entry: %v", err)
			}
		}
	}
}

func (l *Logger) Info(v ...interface{}) {
	if l.isLevelEnabled(levelInfo) {
		if err := l.info.Output(2, fmt.Sprint(v...)); err != nil {
			log.Printf("logger: failed to write info log entry: %v", err)
		}
	}
}

func Info(ctx context.Context, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelInfo) {
			if err := l.info.Output(2, fmt.Sprint(v...)); err != nil {
				log.Printf("logger: failed to write info log entry: %v", err)
			}
		}
	}
}

func (l *Logger) Infof(format string, v ...interface{}) {
	if l.isLevelEnabled(levelInfo) {
		if err := l.info.Output(2, fmt.Sprintf(format, v...)); err != nil {
			log.Printf("logger: failed to write infof log entry: %v", err)
		}
	}
}

func Infof(ctx context.Context, format string, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelInfo) {
			if err := l.info.Output(2, fmt.Sprintf(format, v...)); err != nil {
				log.Printf("logger: failed to write infof log entry: %v", err)
			}
		}
	}
}

// Print / Printf is a compatibility func that behaves like log.Print.

func (l *Logger) Print(v ...interface{}) {
	if l.isLevelEnabled(levelInfo) {
		if err := l.info.Output(2, fmt.Sprint(v...)); err != nil {
			log.Printf("logger: failed to write print log entry: %v", err)
		}
	}
}

func Print(ctx context.Context, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelInfo) {
			if err := l.info.Output(2, fmt.Sprint(v...)); err != nil {
				log.Printf("logger: failed to write print log entry: %v", err)
			}
		}
	}
}

func (l *Logger) Printf(format string, v ...interface{}) {
	if l.isLevelEnabled(levelInfo) {
		if err := l.info.Output(2, fmt.Sprintf(format, v...)); err != nil {
			log.Printf("logger: failed to write print log entry: %v", err)
		}
	}
}

func Printf(ctx context.Context, format string, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelInfo) {
			if err := l.info.Output(2, fmt.Sprintf(format, v...)); err != nil {
				log.Printf("logger: failed to write print log entry: %v", err)
			}
		}
	}
}

func (l *Logger) Warn(v ...interface{}) {
	if l.isLevelEnabled(levelWarn) {
		if err := l.warn.Output(2, fmt.Sprint(v...)); err != nil {
			log.Printf("logger: failed to write warn log entry: %v", err)
		}
	}
}

func Warn(ctx context.Context, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelWarn) {
			if err := l.warn.Output(2, fmt.Sprint(v...)); err != nil {
				log.Printf("logger: failed to write warn log entry: %v", err)
			}
		}
	}
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.isLevelEnabled(levelWarn) {
		if err := l.warn.Output(2, fmt.Sprintf(format, v...)); err != nil {
			log.Printf("logger: failed to write warnf log entry: %v", err)
		}
	}
}

func Warnf(ctx context.Context, format string, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelWarn) {
			if err := l.warn.Output(2, fmt.Sprintf(format, v...)); err != nil {
				log.Printf("logger: failed to write warnf log entry: %v", err)
			}
		}
	}
}

func (l *Logger) Error(v ...interface{}) {
	if l.isLevelEnabled(levelError) {
		if err := l.error.Output(2, fmt.Sprint(v...)); err != nil {
			log.Printf("logger: failed to write error log entry: %v", err)
		}
	}
}

func Error(ctx context.Context, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelError) {
			if err := l.error.Output(2, fmt.Sprint(v...)); err != nil {
				log.Printf("logger: failed to write error log entry: %v", err)
			}
		}
	}
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.isLevelEnabled(levelError) {
		if err := l.error.Output(2, fmt.Sprintf(format, v...)); err != nil {
			log.Printf("logger: failed to write errorf log entry: %v", err)
		}
	}
}

func Errorf(ctx context.Context, format string, v ...interface{}) {
	if l := FromContext(ctx); l != nil {
		if l.isLevelEnabled(levelError) {
			if err := l.error.Output(2, fmt.Sprintf(format, v...)); err != nil {
				log.Printf("logger: failed to write errorf log entry: %v", err)
			}
		}
	}
}

func (l *Logger) IsClosed() bool {
	return l.closed.Load() == 1
}

// SetFlags sets the flags for all loggers.
// debugFlag and stdFlag are the flags from std lib log package.
func (l *Logger) SetFlags(debugFlag, stdFlag int) {
	l.debug.SetFlags(debugFlag)
	l.info.SetFlags(stdFlag)
	l.warn.SetFlags(stdFlag)
	l.error.SetFlags(stdFlag)
}

// SetLevel sets the minimum log level to output.
// Levels are: debug, info, warn, error, none (case-insensitive)
func (l *Logger) SetLevel(level string) error {
	l.closeMu.Lock()
	defer l.closeMu.Unlock()
	if l.IsClosed() {
		return ErrClosed
	}
	var newLevel uint32
	switch strings.ToLower(level) {
	case "debug":
		newLevel = uint32(levelDebug)
		l.debug.SetOutput(l.writer)
		l.info.SetOutput(l.writer)
		l.warn.SetOutput(l.writer)
		l.error.SetOutput(l.writer)
	case "info":
		newLevel = uint32(levelInfo)
		l.debug.SetOutput(io.Discard)
		l.info.SetOutput(l.writer)
		l.warn.SetOutput(l.writer)
		l.error.SetOutput(l.writer)
	case "warn":
		newLevel = uint32(levelWarn)
		l.debug.SetOutput(io.Discard)
		l.info.SetOutput(io.Discard)
		l.warn.SetOutput(l.writer)
		l.error.SetOutput(l.writer)
	case "error":
		newLevel = uint32(levelError)
		l.debug.SetOutput(io.Discard)
		l.info.SetOutput(io.Discard)
		l.warn.SetOutput(io.Discard)
		l.error.SetOutput(l.writer)
	case "none":
		newLevel = uint32(levelNone)
		l.debug.SetOutput(io.Discard)
		l.info.SetOutput(io.Discard)
		l.warn.SetOutput(io.Discard)
		l.error.SetOutput(io.Discard)
	default:
		return fmt.Errorf("invalid log level: '%s'. Valid levels are: debug, info, warn, error, none. %w", level, ErrInvalidLogLevel)
	}
	l.level.Store(newLevel)
	return nil
}

func (l *Logger) Flush() error {
	l.closeMu.Lock()
	defer l.closeMu.Unlock()
	if l.IsClosed() {
		return ErrClosed
	}
	if err := l.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush rlog writer: %w", err)
	}
	return nil
}

func (l *Logger) Close() error {
	l.closeMu.Lock()
	defer l.closeMu.Unlock()
	if l.IsClosed() {
		return ErrClosed
	}
	l.closed.Store(1)
	l.debug.SetOutput(io.Discard)
	l.info.SetOutput(io.Discard)
	l.warn.SetOutput(io.Discard)
	l.error.SetOutput(io.Discard)
	if l.writer != nil {
		err := l.writer.Close()
		l.writer = nil
		if err != nil {
			return fmt.Errorf("failed to close rlog writer: %w", err)
		}
	}
	return nil
}
