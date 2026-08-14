// Package repository provides data-access interfaces and SQLite implementations.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// User is a persisted user row (no password in API responses — hash stays here).
type User struct {
	ID           int64
	Username     string
	PasswordHash string
}

// UserRepository loads and creates users.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	Count(ctx context.Context) (int, error)
	Create(ctx context.Context, username, passwordHash string) (int64, error)
}

// SessionRepository creates and validates sessions.
type SessionRepository interface {
	Create(ctx context.Context, userID int64, expiresAt time.Time) (sessionID string, err error)
	UserIDBySession(ctx context.Context, sessionID string) (userID int64, err error)
	Delete(ctx context.Context, sessionID string) error
}

// SQLiteUserRepository implements UserRepository.
type SQLiteUserRepository struct {
	db *sql.DB
}

// NewSQLiteUserRepository returns a UserRepository backed by db.
func NewSQLiteUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLiteUserRepository) FindByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLiteUserRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (r *SQLiteUserRepository) Create(ctx context.Context, username, passwordHash string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash) VALUES (?, ?)`, username, passwordHash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
