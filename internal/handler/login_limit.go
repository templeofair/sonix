package handler

import (
	"net"
	"sync"
	"time"
)

const (
	loginWindow   = time.Minute
	loginMaxPerIP = 8
)

type loginLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{hits: make(map[string][]time.Time)}
}

func (l *loginLimiter) allow(ip string) bool {
	if ip == "" {
		ip = "unknown"
	}
	now := time.Now()
	cutoff := now.Add(-loginWindow)
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
	if len(hits) >= loginMaxPerIP {
		l.hits[ip] = hits
		return false
	}
	l.hits[ip] = append(hits, now)
	return true
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
