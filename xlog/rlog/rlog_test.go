package rlog

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- constructor -----------------------------------------------------------

// TestNewWriterMissingDirPath verifies that an empty DirPath is rejected.
func TestNewWriterMissingDirPath(t *testing.T) {
	if _, err := NewWriter(Config{}); err == nil {
		t.Fatalf("expected error for empty DirPath, got nil")
	}
}

// --- buffered writes & flush -----------------------------------------------

func TestWriteFlush(t *testing.T) {
	tempDir := t.TempDir()
	w, err := NewWriter(Config{DirPath: tempDir})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	msg := "Hello, test log!\n"
	if n, err := w.Write([]byte(msg)); err != nil || n != len(msg) {
		t.Fatalf("Write: n=%d err=%v", n, err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(tempDir, "latest.log"))
	if string(got) != msg {
		t.Errorf("log mismatch: got %q want %q", got, msg)
	}
}

// --- size-based rotation ----------------------------------------------------

func TestRotation(t *testing.T) {
	tempDir := t.TempDir()
	w, err := NewWriter(Config{DirPath: tempDir, MaxFileSize: 10})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	initial := "abc"    // 3 bytes
	rotate := "defghij" // +7 => 10 bytes total
	_, _ = w.Write([]byte(initial))
	_ = w.Flush()
	_, _ = w.Write([]byte(rotate))
	_ = w.Flush()

	// latest.log must hold only the post-rotation text
	if data, _ := os.ReadFile(filepath.Join(tempDir, "latest.log")); string(data) != rotate {
		t.Errorf("latest.log mismatch")
	}

	// one rotated file must contain the initial payload
	var found bool
	entries, _ := os.ReadDir(tempDir)
	for _, e := range entries {
		if e.Name() == "latest.log" || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		found = true
		if data, _ := os.ReadFile(filepath.Join(tempDir, e.Name())); string(data) != initial {
			t.Errorf("%s mismatch", e.Name())
		}
	}
	if !found {
		t.Errorf("expected a rotated log file")
	}
}

// --- concurrency ------------------------------------------------------------

func TestConcurrentWrites(t *testing.T) {
	tempDir := t.TempDir()
	w, err := NewWriter(Config{DirPath: tempDir})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	const (
		goroutines = 5
		perGo      = 20
	)
	msg := []byte("concurrent\n")

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGo {
				if _, err := w.Write(msg); err != nil {
					t.Errorf("Write: %v", err)
				}
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
	_ = w.Flush()

	got, _ := os.ReadFile(filepath.Join(tempDir, "latest.log"))
	want := goroutines * perGo * len(msg)
	if len(got) != want {
		t.Errorf("bytes written: got %d want %d", len(got), want)
	}
}
