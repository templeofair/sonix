package auth

import (
	"context"
	"database/sql"
	"time"
)

// DBStore implements SessionStore using the database.
// Prefer repository.SessionRepository for new code; kept for SessionStore callers.
type DBStore struct {
	db *sql.DB
}

// NewDBStore returns a new DBStore.
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// CreateSession inserts a session and returns its ID.
func (s *DBStore) CreateSession(ctx context.Context, userID int64, expiresAt time.Time) (string, error) {
	id, err := RandomSessionID()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expiresAt.Format(time.RFC3339))
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetUserIDBySession returns the user ID for a valid session, or 0 and no error if not found/expired.
func (s *DBStore) GetUserIDBySession(ctx context.Context, sessionID string) (int64, error) {
	var userID int64
	var expiresAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM sessions WHERE id = ?`, sessionID).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	t, _ := time.Parse(time.RFC3339, expiresAt)
	if time.Now().After(t) {
		_ = s.DeleteSession(ctx, sessionID)
		return 0, nil
	}
	return userID, nil
}

// DeleteSession removes a session.
func (s *DBStore) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}
