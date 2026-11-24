package httpapi

import (
	"net"
	"net/netip"
	"sync"
	"time"
)

const maxTrackedClients = 10000

type rateLimiter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	now         func() time.Time
	windowStart time.Time
	counts      map[netip.Prefix]int
}

func newRateLimiter(limit int, window time.Duration, now func() time.Time) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, now: now, windowStart: now(), counts: map[netip.Prefix]int{}}
}

func (l *rateLimiter) allow(remoteAddress string) bool {
	client := clientPrefix(remoteAddress)

	l.mu.Lock()
	defer l.mu.Unlock()

	if current := l.now(); current.Sub(l.windowStart) >= l.window {
		l.windowStart = current
		clear(l.counts)
	}
	count, tracked := l.counts[client]
	if !tracked && len(l.counts) >= maxTrackedClients {
		return false
	}
	if count >= l.limit {
		return false
	}
	l.counts[client] = count + 1
	return true
}

func clientPrefix(remoteAddress string) netip.Prefix {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		host = remoteAddress
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Prefix{}
	}
	address = address.Unmap().WithZone("")
	bits := 32
	if address.Is6() {
		bits = 64
	}
	prefix, err := address.Prefix(bits)
	if err != nil {
		return netip.Prefix{}
	}
	return prefix
}
