// Package service holds application use cases (business logic).
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/templeofair/sonix/internal/auth"
	"github.com/templeofair/sonix/internal/repository"
)

var (
	// ErrInvalidCredentials is returned when username/password do not match.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUnauthorized is returned when the session is missing or invalid.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrUserNotFound is returned when the user id does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrSeedPasswordWeak is returned when SEED_PASSWORD is too short.
	ErrSeedPasswordWeak = errors.New("SEED_PASSWORD must be at least 12 characters")
)

// AuthService orchestrates login, logout, and current-user lookup.
type AuthService struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
}

// NewAuthService wires user and session repositories.
func NewAuthService(users repository.UserRepository, sessions repository.SessionRepository) *AuthService {
	return &AuthService{users: users, sessions: sessions}
}

// LoginResult is returned on successful login (session cookie value + max age).
type LoginResult struct {
	SessionID string
	MaxAge    int
}

// Login verifies credentials and creates a session. Does not touch HTTP.
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	// Detach from client cancel so an aborted HTTP request cannot leave the
	// SQLite connection permanently interrupted mid-login.
	dbCtx := context.WithoutCancel(ctx)

	u, err := findUserRetry(dbCtx, s.users, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			auth.BurnLoginTiming(password)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := auth.VerifyPassword(password, u.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}
	expires := time.Now().Add(auth.SessionDuration)
	sessionID, err := createSessionRetry(dbCtx, s.sessions, u.ID, expires)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		SessionID: sessionID,
		MaxAge:    int(auth.SessionDuration.Seconds()),
	}, nil
}

func findUserRetry(ctx context.Context, users repository.UserRepository, username string) (*repository.User, error) {
	var u *repository.User
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		u, err = users.FindByUsername(ctx, username)
		if err == nil || !isTransientSQLite(err) {
			return u, err
		}
		time.Sleep(time.Duration(25*(attempt+1)) * time.Millisecond)
	}
	return u, err
}

func createSessionRetry(ctx context.Context, sessions repository.SessionRepository, userID int64, expires time.Time) (string, error) {
	var sessionID string
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		sessionID, err = sessions.Create(ctx, userID, expires)
		if err == nil || !isTransientSQLite(err) {
			return sessionID, err
		}
		time.Sleep(time.Duration(25*(attempt+1)) * time.Millisecond)
	}
	return sessionID, err
}

// Logout deletes the session if present. Always succeeds from a business view.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.sessions.Delete(ctx, sessionID)
}

// Me returns the username for a logged-in user id.
func (s *AuthService) Me(ctx context.Context, userID int64) (username string, err error) {
	if userID == 0 {
		return "", ErrUnauthorized
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrUserNotFound
		}
		return "", err
	}
	return u.Username, nil
}

// UserIDFromSession resolves a session cookie value to a user id.
func (s *AuthService) UserIDFromSession(ctx context.Context, sessionID string) (int64, error) {
	if sessionID == "" {
		return 0, ErrUnauthorized
	}
	userID, err := s.sessions.UserIDBySession(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	if userID == 0 {
		return 0, ErrUnauthorized
	}
	return userID, nil
}

func isTransientSQLite(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "interrupted") ||
		strings.Contains(msg, "busy") ||
		strings.Contains(msg, "locked") ||
		strings.Contains(msg, "database is locked")
}

// SeedUserIfEmpty creates the first user when the table is empty (startup seed).
func (s *AuthService) SeedUserIfEmpty(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return nil
	}
	n, err := s.users.Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if len(password) < 12 {
		return ErrSeedPasswordWeak
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.users.Create(ctx, username, hash)
	return err
}
