package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// SessionStore creates and validates sessions (e.g. in DB).
type SessionStore interface {
	CreateSession(ctx context.Context, userID int64, expiresAt time.Time) (sessionID string, err error)
	GetUserIDBySession(ctx context.Context, sessionID string) (userID int64, err error)
	DeleteSession(ctx context.Context, sessionID string) error
}

// CookieName is the name of the session cookie.
const CookieName = "sonix_session"

// SecureCookie should be true in production (HTTPS).
var SecureCookie = false

// SessionDuration is how long a session lasts.
const SessionDuration = 24 * time.Hour

// RandomSessionID returns a new random session ID.
func RandomSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
