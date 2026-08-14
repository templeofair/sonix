package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/templeofair/sonix/internal/auth"
	"github.com/templeofair/sonix/internal/repository"
	"github.com/templeofair/sonix/internal/service"
)

type mockUserRepo struct {
	byUsername map[string]*repository.User
	byID       map[int64]*repository.User
	count      int
	createErr  error
}

func (m *mockUserRepo) FindByUsername(_ context.Context, username string) (*repository.User, error) {
	u, ok := m.byUsername[username]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id int64) (*repository.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) Count(context.Context) (int, error) { return m.count, nil }

func (m *mockUserRepo) Create(_ context.Context, username, passwordHash string) (int64, error) {
	if m.createErr != nil {
		return 0, m.createErr
	}
	id := int64(len(m.byID) + 1)
	u := &repository.User{ID: id, Username: username, PasswordHash: passwordHash}
	if m.byUsername == nil {
		m.byUsername = map[string]*repository.User{}
	}
	if m.byID == nil {
		m.byID = map[int64]*repository.User{}
	}
	m.byUsername[username] = u
	m.byID[id] = u
	m.count++
	return id, nil
}

type mockSessionRepo struct {
	sessions map[string]int64
	createID string
}

func (m *mockSessionRepo) Create(_ context.Context, userID int64, _ time.Time) (string, error) {
	id := m.createID
	if id == "" {
		id = "sess-1"
	}
	if m.sessions == nil {
		m.sessions = map[string]int64{}
	}
	m.sessions[id] = userID
	return id, nil
}

func (m *mockSessionRepo) UserIDBySession(_ context.Context, sessionID string) (int64, error) {
	id, ok := m.sessions[sessionID]
	if !ok {
		return 0, nil
	}
	return id, nil
}

func (m *mockSessionRepo) Delete(_ context.Context, sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

func TestAuthService_Login_Success(t *testing.T) {
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	users := &mockUserRepo{
		byUsername: map[string]*repository.User{
			"alice": {ID: 7, Username: "alice", PasswordHash: hash},
		},
	}
	sessions := &mockSessionRepo{createID: "abc"}
	svc := service.NewAuthService(users, sessions)

	res, err := svc.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if res.SessionID != "abc" {
		t.Fatalf("session id = %q", res.SessionID)
	}
	if sessions.sessions["abc"] != 7 {
		t.Fatalf("session user = %d", sessions.sessions["abc"])
	}
}

func TestAuthService_Login_BadPassword(t *testing.T) {
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	users := &mockUserRepo{
		byUsername: map[string]*repository.User{
			"alice": {ID: 7, Username: "alice", PasswordHash: hash},
		},
	}
	svc := service.NewAuthService(users, &mockSessionRepo{})

	_, err = svc.Login(context.Background(), "alice", "wrong")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthService_Login_UnknownUser(t *testing.T) {
	svc := service.NewAuthService(&mockUserRepo{}, &mockSessionRepo{})
	_, err := svc.Login(context.Background(), "nobody", "x")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthService_Me(t *testing.T) {
	users := &mockUserRepo{
		byID: map[int64]*repository.User{
			3: {ID: 3, Username: "bob", PasswordHash: "x"},
		},
	}
	svc := service.NewAuthService(users, &mockSessionRepo{})
	name, err := svc.Me(context.Background(), 3)
	if err != nil || name != "bob" {
		t.Fatalf("Me = %q, %v", name, err)
	}
	_, err = svc.Me(context.Background(), 99)
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestAuthService_UserIDFromSession(t *testing.T) {
	sessions := &mockSessionRepo{sessions: map[string]int64{"ok": 5}}
	svc := service.NewAuthService(&mockUserRepo{}, sessions)
	id, err := svc.UserIDFromSession(context.Background(), "ok")
	if err != nil || id != 5 {
		t.Fatalf("got %d %v", id, err)
	}
	_, err = svc.UserIDFromSession(context.Background(), "missing")
	if !errors.Is(err, service.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestAuthService_SeedUserIfEmpty(t *testing.T) {
	users := &mockUserRepo{count: 0}
	svc := service.NewAuthService(users, &mockSessionRepo{})
	if err := svc.SeedUserIfEmpty(context.Background(), "admin", "test-password-1"); err != nil {
		t.Fatal(err)
	}
	if users.count != 1 || users.byUsername["admin"] == nil {
		t.Fatalf("user not created: %+v", users)
	}
	// second seed should no-op when count > 0
	if err := svc.SeedUserIfEmpty(context.Background(), "other", "test-password-1"); err != nil {
		t.Fatal(err)
	}
	if users.count != 1 {
		t.Fatalf("count = %d", users.count)
	}
	if err := svc.SeedUserIfEmpty(context.Background(), "admin", "short"); err != nil {
		t.Fatalf("existing users ignore short seed: %v", err)
	}
}

func TestAuthService_SeedPasswordWeak(t *testing.T) {
	users := &mockUserRepo{count: 0}
	svc := service.NewAuthService(users, &mockSessionRepo{})
	if err := svc.SeedUserIfEmpty(context.Background(), "admin", "short"); !errors.Is(err, service.ErrSeedPasswordWeak) {
		t.Fatalf("short seed: %v", err)
	}
	if users.count != 0 {
		t.Fatal("must not create user")
	}
}
