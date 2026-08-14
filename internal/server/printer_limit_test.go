package server

import "testing"

func TestPrinterLimiter_PerIP(t *testing.T) {
	l := newPrinterLimiter()
	for i := 0; i < printerMaxPerIP; i++ {
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
