package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

// SQLiteSessionRepository implements SessionRepository using the sessions table.
type SQLiteSessionRepository struct {
	db *sql.DB
}

// NewSQLiteSessionRepository returns a SessionRepository backed by db.
func NewSQLiteSessionRepository(db *sql.DB) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{db: db}
}

func randomSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (r *SQLiteSessionRepository) Create(ctx context.Context, userID int64, expiresAt time.Time) (string, error) {
	id, err := randomSessionID()
	if err != nil {
		return "", err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expiresAt.Format(time.RFC3339))
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *SQLiteSessionRepository) UserIDBySession(ctx context.Context, sessionID string) (int64, error) {
	var userID int64
	var expiresAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM sessions WHERE id = ?`, sessionID).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	t, _ := time.Parse(time.RFC3339, expiresAt)
	if time.Now().After(t) {
		_ = r.Delete(ctx, sessionID)
		return 0, nil
	}
	return userID, nil
}

func (r *SQLiteSessionRepository) Delete(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}
