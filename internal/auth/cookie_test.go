package auth

import "testing"

func TestValidateSessionSecret(t *testing.T) {
	if err := ValidateSessionSecret(""); err != ErrSessionSecretRequired {
		t.Fatalf("empty: %v", err)
	}
	if err := ValidateSessionSecret("change-me-in-production"); err != ErrSessionSecretWeak {
		t.Fatalf("placeholder: %v", err)
	}
	if err := ValidateSessionSecret("short"); err != ErrSessionSecretWeak {
		t.Fatalf("short: %v", err)
	}
	if err := ValidateSessionSecret("test-secret-at-least-16"); err != nil {
		t.Fatalf("ok: %v", err)
	}
}

func TestCookieMAC_RoundTripAndTamper(t *testing.T) {
	m, err := NewCookieMAC("test-secret-at-least-16")
	if err != nil {
		t.Fatal(err)
	}
	id := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	val := m.Sign(id)
	got, ok := m.Parse(val)
	if !ok || got != id {
		t.Fatalf("roundtrip got %q ok=%v", got, ok)
	}
	if _, ok := m.Parse(id); ok {
		t.Fatal("unsigned session id must not parse")
	}
	if _, ok := m.Parse(val[:len(val)-1] + "0"); ok {
		t.Fatal("tampered mac must not parse")
	}
	other, _ := NewCookieMAC("other-secret-at-least16")
	if _, ok := other.Parse(val); ok {
		t.Fatal("wrong secret must not parse")
	}
}
