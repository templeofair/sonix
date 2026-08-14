package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	// MinSessionSecretLen is the shortest SESSION_SECRET we accept.
	MinSessionSecretLen = 16
	cookieSep           = "."
)

var (
	// ErrSessionSecretRequired is returned when SESSION_SECRET is empty.
	ErrSessionSecretRequired = errors.New("required; generate one with: openssl rand -hex 32")
	// ErrSessionSecretWeak is returned for published placeholders or short values.
	ErrSessionSecretWeak = errors.New("too weak; generate one with: openssl rand -hex 32")
)

// CookieMAC HMAC-SHA256-signs session IDs for the session cookie.
type CookieMAC struct {
	secret []byte
}

// ValidateSessionSecret rejects empty, placeholder, and short secrets.
func ValidateSessionSecret(secret string) error {
	s := strings.TrimSpace(secret)
	if s == "" {
		return ErrSessionSecretRequired
	}
	switch strings.ToLower(s) {
	case "change-me-in-production", "your-secret", "secret", "password":
		return ErrSessionSecretWeak
	}
	if len(s) < MinSessionSecretLen {
		return ErrSessionSecretWeak
	}
	return nil
}

// NewCookieMAC returns a signer for SESSION_SECRET.
func NewCookieMAC(secret string) (*CookieMAC, error) {
	if err := ValidateSessionSecret(secret); err != nil {
		return nil, err
	}
	return &CookieMAC{secret: []byte(strings.TrimSpace(secret))}, nil
}

// Sign returns cookie value sessionID.hexmac.
func (m *CookieMAC) Sign(sessionID string) string {
	if m == nil || sessionID == "" {
		return ""
	}
	return sessionID + cookieSep + m.macHex(sessionID)
}

// Parse verifies the cookie and returns the raw session ID.
func (m *CookieMAC) Parse(value string) (sessionID string, ok bool) {
	if m == nil || value == "" {
		return "", false
	}
	id, mac, found := strings.Cut(value, cookieSep)
	if !found || id == "" || mac == "" {
		return "", false
	}
	if strings.Contains(mac, cookieSep) {
		return "", false
	}
	want, err := hex.DecodeString(mac)
	if err != nil || len(want) != sha256.Size {
		return "", false
	}
	got, err := hex.DecodeString(m.macHex(id))
	if err != nil {
		return "", false
	}
	if !hmac.Equal(got, want) {
		return "", false
	}
	return id, true
}

func (m *CookieMAC) macHex(sessionID string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(sessionID))
	return hex.EncodeToString(mac.Sum(nil))
}
