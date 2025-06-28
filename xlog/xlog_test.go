package xlog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Data-Corruption/stdx/xlog"
)

// quick and non-exhaustive, i'll add proper tests later

func TestNewInvalidLevel(t *testing.T) {
	_, err := xlog.New(t.TempDir(), "bogus")
	if err == nil {
		t.Fatalf("expected error for invalid level")
	}
}

func TestIntoFromContextRoundTrip(t *testing.T) {
	l, err := xlog.New(t.TempDir(), "debug")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	ctx := xlog.IntoContext(context.Background(), l)
	if got := xlog.FromContext(ctx); got != l {
		t.Fatalf("round-trip mismatch: want %p, got %p", l, got)
	}
}

func TestCloseIdempotentAndLocked(t *testing.T) {
	l, _ := xlog.New(t.TempDir(), "info")

	if err := l.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := l.Close(); !errors.Is(err, xlog.ErrClosed) {
		t.Fatalf("second close should return ErrClosed, got %v", err)
	}
}

func TestSetLevelAndFlushAfterClose(t *testing.T) {
	l, _ := xlog.New(t.TempDir(), "warn")
	_ = l.Close()

	if err := l.SetLevel("debug"); !errors.Is(err, xlog.ErrClosed) {
		t.Fatalf("SetLevel after close: want ErrClosed, got %v", err)
	}
	if err := l.Flush(); !errors.Is(err, xlog.ErrClosed) {
		t.Fatalf("Flush after close: want ErrClosed, got %v", err)
	}
}
