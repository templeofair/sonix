package handler

import (
	"testing"
	"time"
)

func TestLoginLimiter_PerIP(t *testing.T) {
	l := newLoginLimiter()
	for i := 0; i < loginMaxPerIP; i++ {
		if !l.allow("203.0.113.10") {
			t.Fatalf("attempt %d denied", i+1)
		}
	}
	if l.allow("203.0.113.10") {
		t.Fatal("expected deny after max")
	}
	if !l.allow("203.0.113.11") {
		t.Fatal("other IP should be allowed")
	}
}

func TestLoginLimiter_WindowExpiry(t *testing.T) {
	l := newLoginLimiter()
	old := time.Now().Add(-loginWindow - time.Second)
	l.hits["203.0.113.10"] = make([]time.Time, loginMaxPerIP)
	for i := range l.hits["203.0.113.10"] {
		l.hits["203.0.113.10"][i] = old
	}
	if !l.allow("203.0.113.10") {
		t.Fatal("expired hits should not count")
	}
}
