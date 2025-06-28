package xhttp

import (
	"net/http"
	"testing"
	"time"
)

func noopHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})
}

func TestNewServerValidation(t *testing.T) {
	t.Run("nil handler", func(t *testing.T) {
		_, err := NewServer(&ServerConfig{})
		if err == nil {
			t.Fatalf("expected error when Handler is nil")
		}
	})

	t.Run("tls without cert paths", func(t *testing.T) {
		_, err := NewServer(&ServerConfig{
			UseTLS:  true,
			Handler: noopHandler(),
		})
		if err == nil {
			t.Fatalf("expected error when TLS paths are missing")
		}
	})
}

func TestNewServerDefaultsApplied(t *testing.T) {
	srv, err := NewServer(&ServerConfig{Handler: noopHandler()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := srv.cfg.Addr; got != DefaultAddr {
		t.Errorf("Addr default: want %q, got %q", DefaultAddr, got)
	}
	if got := srv.cfg.ReadTimeout; got != DefaultReadTimeout {
		t.Errorf("ReadTimeout default: want %s, got %s", DefaultReadTimeout, got)
	}
	if got := srv.cfg.WriteTimeout; got != DefaultWriteTimeout {
		t.Errorf("WriteTimeout default: want %s, got %s", DefaultWriteTimeout, got)
	}
	if got := srv.cfg.IdleTimeout; got != DefaultIdleTimeout {
		t.Errorf("IdleTimeout default: want %s, got %s", DefaultIdleTimeout, got)
	}
	if got := srv.cfg.ShutdownTimeout; got != DefaultShutdownTimeout {
		t.Errorf("ShutdownTimeout default: want %s, got %s", DefaultShutdownTimeout, got)
	}
	if got := srv.cfg.AfterListenDelay; got != DefaultAfterListenDelay {
		t.Errorf("AfterListenDelay default: want %s, got %s", DefaultAfterListenDelay, got)
	}

	if srv.server.TLSConfig == nil {
		t.Errorf("TLSConfig should never be nil (even for non-TLS servers)")
	}
}

func TestNewServerTLSAddrDefault(t *testing.T) {
	srv, err := NewServer(&ServerConfig{
		UseTLS:      true,
		TLSKeyPath:  "./testdata/key.pem",
		TLSCertPath: "./testdata/cert.pem",
		Handler:     noopHandler(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := srv.cfg.Addr; got != DefaultTLSAddr {
		t.Errorf("TLS Addr default: want %q, got %q", DefaultTLSAddr, got)
	}
}

func TestServerListenLifecycle(t *testing.T) {
	srv, err := NewServer(&ServerConfig{Handler: noopHandler(), Addr: ":0"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	done := make(chan struct{})
	go func() { _ = srv.Listen(); close(done) }()

	time.Sleep(50 * time.Millisecond) // give the listener time to start
	if srv.server.Addr == "" {
		t.Fatalf("listener did not pick an address")
	}

	_ = srv.server.Close() // trigger graceful shutdown
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not shut down in time")
	}
}
