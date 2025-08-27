package xnet

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestHasUsableAddr(t *testing.T) {
	if !hasUsableAddr() {
		t.Log("no global unicast addr found on this host — test may be running in very restricted env")
	}
}

func TestTCPProbeSuccess(t *testing.T) {
	// Start a local TCP listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	ctx := context.Background()
	if err := tcpProbe(ctx, addr); err != nil {
		t.Fatalf("tcpProbe to %s failed: %v", addr, err)
	}
}

func TestTCPProbeFail(t *testing.T) {
	// Unused port (probabilistic, but high port should be safe)
	addr := "127.0.0.1:65000"
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := tcpProbe(ctx, addr); err == nil {
		t.Fatalf("expected tcpProbe to %s to fail, but it succeeded", addr)
	}
}

func TestDNSProbeLocalhost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := dnsProbe(ctx, "localhost"); err != nil {
		t.Fatalf("dnsProbe for localhost failed: %v", err)
	}
}

func TestWaitWithLocalListener(t *testing.T) {
	// Start a local TCP listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	ctx := context.Background()
	if err := Wait(ctx, 2*time.Second, "tcp:"+addr); err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
}

func TestWaitTimeout(t *testing.T) {
	ctx := context.Background()
	// Port almost certainly closed
	addr := "127.0.0.1:65001"

	start := time.Now()
	err := Wait(ctx, 1*time.Second, "tcp:"+addr)
	if err == nil {
		t.Fatal("expected Wait to fail but it succeeded")
	}
	if time.Since(start) < 900*time.Millisecond {
		t.Fatalf("Wait returned too quickly, expected ~1s timeout")
	}
}
