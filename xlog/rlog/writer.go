// Package rlog offers a small, production-ready log writer that embraces
// stdlib ideals: simple, focused, and composable.
//
// Current extension:
//   - [Writer] implements buffered, size-based log rotation with optional
//     age-based flushing—ideal for long-running services that want durable
//     logs without pulling in a full logging framework.
//
// [Writer] usage:
//
//	w, err := rlog.NewWriter(rlog.Config{
//	  DirPath:     "./logs",        // required - will be created if missing
//	  MaxFileSize: 512 << 20,       // 512 MB before rotation (optional)
//	  MaxBufSize:  8 * 1024,        // 8 KB in-memory buffer    (optional)
//	  MaxBufAge:   5 * time.Second, // flush after 5 s        (optional)
//	})
//	if err != nil {
//	  log.Fatalf("rlog: %v", err)
//	}
//	defer w.Close()
//
//	// plain io.Writer usage
//	log.SetOutput(w)
//	log.Println("hello, rotating world")
//
// Manual flush / error check:
//
//	if err := w.Flush(); err != nil {
//	  log.Printf("flush failed: %v", err)
//	}
//	if err := w.Error(); err != nil {
//	  log.Printf("writer is unhealthy: %v", err)
//	}
//
// Internals & caveats:
//   - Rotation renames the active file to a timestamped `<ts>.log` and
//     re-creates `latest.log` atomically. A lightweight file-lock prevents
//     concurrent rotations across processes.
//   - A single [Writer] should be used per directory per process; multiple
//     processes may safely share the same directory.
package rlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	DefaultMaxFileSize = 256 * 1024 * 1024 // 256 MB
	DefaultMaxBufSize  = 4096              // 4 KB
	DefaultMaxBufAge   = 15 * time.Second  // 15 seconds
)

type noCopy struct{} // see https://github.com/golang/go/issues/8005#issuecomment-190753527

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// Config holds configuration options for [Writer].
type Config struct {
	DirPath     string        // Directory path where log files are stored. Created if it does not exist.
	MaxFileSize int64         // Soft max size of a log file before rotation occurs. Default is 256 MB.
	MaxBufSize  int           // Soft max size of the buffer before flushing to disk. Default is 4 KB.
	MaxBufAge   time.Duration // Max age of the buffer before flushing to disk. Default is 15 seconds. Negative to disable.
}

// Writer implements [io.Writer] for buffered log writing with automatic file rotation.
// If a write operation returns an error, no further data is accepted and subsequent
// function calls will return the error. Same as seen in various standard library packages.
//
// WARNING: Only a single [Writer] should be used per directory per process. Multiple
// process instances writing to the same directory is fine, Multiple [Writer] instances
// within the same process doing so is not.
type Writer struct {
	noCopy noCopy
	mu     sync.Mutex
	err    error
	cfg    Config
	buf    []byte
	file   *os.File
	// closeAgeTrigger is a channel used to clean up the age-triggered flush goroutine.
	closeAgeTrigger chan struct{}
}

// NewWriter creates and initializes a new [Writer] for the specified directory.
// Creating the directory if it does not already exist. Additional options can
// be provided to customize the Writer's behavior.
func NewWriter(cfg Config) (*Writer, error) {
	if cfg.DirPath == "" {
		return nil, fmt.Errorf("directory path must be provided")
	}

	writer := &Writer{cfg: cfg}

	// set defaults
	if cfg.MaxFileSize <= 0 {
		writer.cfg.MaxFileSize = DefaultMaxFileSize
	}
	if cfg.MaxBufSize <= 0 {
		writer.cfg.MaxBufSize = DefaultMaxBufSize
	}
	if cfg.MaxBufAge == 0 { // leave neg for disabling
		writer.cfg.MaxBufAge = DefaultMaxBufAge
	}

	// setup buff
	writer.buf = make([]byte, 0, writer.cfg.MaxBufSize)

	// ensure directory exists
	if err := os.MkdirAll(cfg.DirPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory '%s': %w", cfg.DirPath, err)
	}

	// open latest log file
	var err error
	if writer.file, err = os.OpenFile(writer.latestPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return nil, err
	}

	// start goroutine for age triggered flushes
	d := writer.cfg.MaxBufAge
	if d > 0 {
		writer.closeAgeTrigger = make(chan struct{})
		go func() {
			ticker := time.NewTicker(d)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := writer.Flush(); err != nil {
						return
					}
				case <-writer.closeAgeTrigger:
					return
				}
			}
		}()
	}

	return writer, nil
}

// exported

// Write appends p to [Writer.buf]. If the write would overflow the buffer,
// the current buffer is flushed first. When p itself is ≥ MaxBufSize
// the data is written straight to disk instead of being buffered.
// Returns the length of p on success. Partial writes are not supported.
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.err != nil {
		return 0, w.err
	}
	pLen := len(p)

	// flush if adding p would overflow
	if len(w.buf)+pLen >= w.cfg.MaxBufSize {
		if err := w.flush(); err != nil {
			return 0, err
		}
	}

	// if p ≥ MaxBufSize, stream it directly
	// to the file to avoid an oversized in-memory allocation
	if pLen >= w.cfg.MaxBufSize {
		// correct any rot drift
		if err := w.ensureCurrentFile(); err != nil {
			return 0, err
		}

		// rotate if this write would overflow the file.
		fi, err := w.file.Stat()
		if err != nil {
			w.err = fmt.Errorf("stat log file: %v", err)
			return 0, w.err
		}
		if fi.Size()+int64(pLen) >= w.cfg.MaxFileSize {
			if err := w.rotate(); err != nil {
				return 0, err
			}
		}

		if _, err := w.file.Write(p); err != nil {
			w.err = fmt.Errorf("write log file: %v", err)
			return 0, w.err
		}
		if err := w.file.Sync(); err != nil {
			w.err = fmt.Errorf("sync log file: %v", err)
			return 0, w.err
		}
		return pLen, nil
	}

	// normal case
	w.buf = append(w.buf, p...)
	return pLen, nil
}

// Flush appends [Writer.buf] to 'DirPath/latest.log', rotates first if appending
// would result in latest.log exceeding MaxFileSize, then clears [Writer.buf].
// Returns an error if the write, file sync, or rotation fails.
//
// Flushing happens automatically during [Writer.Write] when [Writer.buf] exceeds MaxBufSize.
func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.flush()
}

// Error returns the last error encountered by the Writer.
// If no error has occurred, it returns nil.
func (w *Writer) Error() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}

// Close flushes the Writer, age trigger goroutine, and open file.
// It should be called when the Writer is no longer needed.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closeAgeTrigger != nil {
		close(w.closeAgeTrigger)
		w.closeAgeTrigger = nil
	}

	if w.err != nil || w.file == nil {
		return w.err
	}
	if err := w.flush(); err != nil {
		return err
	}
	return w.file.Close()
}

// internal

// flush appends [Writer.buf] to 'DirPath/latest.log', rotates first if appending
// would result in latest.log exceeding MaxFileSize, then clears [Writer.buf].
// Returns an error if the write, file sync, or rotation fails. Assumes mutex is held by caller.
func (w *Writer) flush() error {
	if w.err != nil {
		return w.err
	}
	if w.file == nil {
		w.err = fmt.Errorf("log file %q is closed", w.latestPath())
		return w.err
	}
	if len(w.buf) == 0 {
		return nil
	}
	// correct any rot drift
	if err := w.ensureCurrentFile(); err != nil {
		w.err = err
		return err
	}
	// determine if the file needs to be rotated.
	fi, err := w.file.Stat()
	if err != nil {
		w.err = fmt.Errorf("failed to stat log file: %v", err)
		return w.err
	}
	if fi.Size()+int64(len(w.buf)) >= w.cfg.MaxFileSize {
		if err := w.rotate(); err != nil {
			return err
		}
	}
	// write the buffer to the file and sync.
	if _, err := w.file.Write(w.buf); err != nil {
		w.err = fmt.Errorf("failed to write to log file: %v", err)
		return w.err
	}
	if err := w.file.Sync(); err != nil {
		w.err = fmt.Errorf("failed to sync log file: %v", err)
		return w.err
	}
	w.buf = w.buf[:0]
	return nil
}

// rotate renames the latest log file to the current timestamp and creates a
// new "latest.log" file for subsequent writes. Assumes mutex is held by caller.
func (w *Writer) rotate() error {
	if w.err != nil {
		return w.err
	}

	unlock, err := acquireRotationLock(w.cfg.DirPath)
	if err != nil {
		w.err = fmt.Errorf("failed to acquire rotation lock: %v", err)
		return w.err
	}
	if unlock != nil {
		defer unlock()
	}

	if w.file != nil {
		if err := w.file.Close(); err != nil {
			w.err = fmt.Errorf("failed to close log file: %v", err)
			return w.err
		}
		w.file = nil
	}
	oldPath := w.latestPath()
	ts := time.Now().Format("20060102-150405.000000") // sub-second in case of high-frequency rotation
	newPath := filepath.Join(w.cfg.DirPath, fmt.Sprintf("%s.log", ts))
	if err := os.Rename(oldPath, newPath); err != nil {
		w.err = fmt.Errorf("failed to rename log file: %v", err)
		return err
	}
	if w.file, err = os.OpenFile(oldPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		w.err = fmt.Errorf("failed to create new log file: %v", err)
		return err
	}
	return nil
}

// ensureCurrentFile reopens the latest log file if it has been rotated by another process.
func (w *Writer) ensureCurrentFile() error {
	latestInfo, err := os.Stat(w.latestPath())
	if err != nil {
		return err
	}
	currentInfo, err := w.file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(latestInfo, currentInfo) {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file, err = os.OpenFile(w.latestPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	}
	return err
}

// latestPath returns the path to the latest log file.
func (w *Writer) latestPath() string {
	return filepath.Join(w.cfg.DirPath, "latest.log")
}
