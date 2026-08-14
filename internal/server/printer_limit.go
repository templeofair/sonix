package server

import (
	"net"
	"sync"
	"time"
)

const (
	printerWindow   = time.Minute
	printerMaxPerIP = 4
)

type printerLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newPrinterLimiter() *printerLimiter {
	return &printerLimiter{hits: make(map[string][]time.Time)}
}

func (l *printerLimiter) allow(ip string) bool {
	if ip == "" {
		ip = "unknown"
	}
	now := time.Now()
	cutoff := now.Add(-printerWindow)
	l.mu.Lock()
	defer l.mu.Unlock()
	hits := l.hits[ip]
	n := 0
	for _, t := range hits {
		if t.After(cutoff) {
			hits[n] = t
			n++
		}
	}
	hits = hits[:n]
	if len(hits) >= printerMaxPerIP {
		l.hits[ip] = hits
		return false
	}
	l.hits[ip] = append(hits, now)
	return true
}

func printerClientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
