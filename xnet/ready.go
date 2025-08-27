package xnet

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"net"
	"time"
)

// Wait blocks until "the network is probably usable" or ctx/timeout expires.
// Success = at least one non-loopback, UP iface has a global IP AND at least one probe succeeds.
// Probes are conservative defaults; you can pass alternatives (e.g., "tcp:192.0.2.1:443", "dns:yourdomain.tld").
func Wait(ctx context.Context, timeout time.Duration, probes ...string) error {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if len(probes) == 0 {
		probes = []string{
			"tcp:1.1.1.1:53",                // Cloudflare v4 DNS
			"tcp:[2606:4700:4700::1111]:53", // Cloudflare v6 DNS
			"dns:example.com",               // any resolvable hostname
		}
	}

	// exponential backoff up to ~2s with a bit of jitter
	nextDelay := func(attempt int) time.Duration {
		base := 100 * time.Millisecond
		max := 2 * time.Second
		d := time.Duration(float64(base) * math.Pow(1.7, float64(attempt)))
		if d > max {
			d = max
		}
		// jitter +/- 25%
		j := time.Duration(rand.Int63n(int64(d/2))) - d/4
		return d + j
	}

	for attempt := 0; ; attempt++ {
		if hasUsableAddr() && anyProbeOK(ctx, probes) {
			return nil
		}
		select {
		case <-ctx.Done():
			return context.DeadlineExceeded
		case <-time.After(nextDelay(attempt)):
		}
	}
}

func hasUsableAddr() bool {
	ifis, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, ifi := range ifis {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.IsGlobalUnicast() {
				return true
			}
		}
	}
	return false
}

func anyProbeOK(parent context.Context, probes []string) bool {
	for _, p := range probes {
		switch {
		case len(p) > 4 && p[:4] == "tcp:":
			if tcpProbe(parent, p[4:]) == nil {
				return true
			}
		case len(p) > 4 && p[:4] == "dns:":
			if dnsProbe(parent, p[4:]) == nil {
				return true
			}
		}
	}
	return false
}

func tcpProbe(parent context.Context, addr string) error {
	ctx, cancel := context.WithTimeout(parent, 1*time.Second)
	defer cancel()
	var d net.Dialer
	c, err := d.DialContext(ctx, "tcp", addr)
	if err == nil {
		_ = c.Close()
		return nil
	}
	// treat "context deadline exceeded" as failure; everything else is fine to retry
	return err
}

func dnsProbe(parent context.Context, name string) error {
	ctx, cancel := context.WithTimeout(parent, 1*time.Second)
	defer cancel()
	r := &net.Resolver{}
	ips, err := r.LookupIPAddr(ctx, name)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return errors.New("no A/AAAA records")
	}
	return nil
}
