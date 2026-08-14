package handler_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/templeofair/sonix/internal/auth"
	"github.com/templeofair/sonix/internal/database"
	"github.com/templeofair/sonix/internal/handler"
	"github.com/templeofair/sonix/internal/repository"
	"github.com/templeofair/sonix/internal/service"
)

func newAuthHandler(t *testing.T) (*handler.Auth, *sqlDB) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "auth.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	users := repository.NewSQLiteUserRepository(db)
	sessions := repository.NewSQLiteSessionRepository(db)
	svc := service.NewAuthService(users, sessions)
	mac, err := auth.NewCookieMAC("test-secret-at-least-16")
	if err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword("pw")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := users.Create(context.Background(), "tester", hash); err != nil {
		t.Fatal(err)
	}
	return handler.NewAuth(svc, mac), &sqlDB{db: db, users: users, sessions: sessions}
}

// sqlDB is only for typing cleanup; tests use handler HTTP surface.
type sqlDB struct {
	db       interface{ Close() error }
	users    repository.UserRepository
	sessions repository.SessionRepository
}

func TestAuthHandler_LoginLogoutMe(t *testing.T) {
	h, _ := newAuthHandler(t)

	body, _ := json.Marshal(map[string]string{"username": "tester", "password": "pw"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login status %d body %s", rr.Code, rr.Body.String())
	}
	var loginResp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&loginResp); err != nil || loginResp["ok"] != "true" {
		t.Fatalf("login body: %v %v", loginResp, err)
	}
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("missing session cookie")
	}
	if sessionCookie.Secure {
		t.Fatal("HTTP login must not set Secure (LAN still uses HTTP)")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq.AddCookie(sessionCookie)
	meRR := httptest.NewRecorder()
	h.RequireAuth(h.Me)(meRR, meReq)
	if meRR.Code != http.StatusOK {
		t.Fatalf("me status %d", meRR.Code)
	}
	var me map[string]string
	_ = json.NewDecoder(meRR.Body).Decode(&me)
	if me["username"] != "tester" {
		t.Fatalf("me = %#v", me)
	}

	outReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	outReq.AddCookie(sessionCookie)
	outRR := httptest.NewRecorder()
	h.Logout(outRR, outReq)
	if outRR.Code != http.StatusOK {
		t.Fatalf("logout %d", outRR.Code)
	}

	meReq2 := httptest.NewRequest(http.MethodGet, "/me", nil)
	meReq2.AddCookie(sessionCookie)
	meRR2 := httptest.NewRecorder()
	h.RequireAuth(h.Me)(meRR2, meReq2)
	if meRR2.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status %d", meRR2.Code)
	}

	tamper := *sessionCookie
	tamper.Value = sessionCookie.Value + "a"
	badReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	badReq.AddCookie(&tamper)
	badRR := httptest.NewRecorder()
	h.RequireAuth(h.Me)(badRR, badReq)
	if badRR.Code != http.StatusUnauthorized {
		t.Fatalf("tampered cookie status %d", badRR.Code)
	}
}

func TestAuthHandler_LoginCookieSecureOnTLS(t *testing.T) {
	h, _ := newAuthHandler(t)
	body, _ := json.Marshal(map[string]string{"username": "tester", "password": "pw"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name != auth.CookieName {
			continue
		}
		found = true
		if !c.Secure {
			t.Fatal("TLS login must set Secure")
		}
	}
	if !found {
		t.Fatal("missing session cookie")
	}
}

func TestAuthHandler_LoginRateLimit(t *testing.T) {
	h, _ := newAuthHandler(t)
	body, _ := json.Marshal(map[string]string{"username": "tester", "password": "nope"})
	var last int
	for i := 0; i < 9; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.RemoteAddr = "203.0.113.10:12345"
		rr := httptest.NewRecorder()
		h.Login(rr, req)
		last = rr.Code
		if i < 8 && rr.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status %d", i+1, rr.Code)
		}
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("9th status %d", last)
	}
	other := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	other.RemoteAddr = "203.0.113.11:12345"
	otherRR := httptest.NewRecorder()
	h.Login(otherRR, other)
	if otherRR.Code != http.StatusUnauthorized {
		t.Fatalf("other IP status %d", otherRR.Code)
	}
}

func TestAuthHandler_LoginBadCreds(t *testing.T) {
	h, _ := newAuthHandler(t)
	body, _ := json.Marshal(map[string]string{"username": "tester", "password": "nope"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestAuthHandler_LoginValidation(t *testing.T) {
	h, _ := newAuthHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestSQLiteSession_Expired(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sess.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	users := repository.NewSQLiteUserRepository(db)
	sessions := repository.NewSQLiteSessionRepository(db)
	hash, _ := auth.HashPassword("x")
	uid, err := users.Create(context.Background(), "u", hash)
	if err != nil {
		t.Fatal(err)
	}
	sid, err := sessions.Create(context.Background(), uid, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	got, err := sessions.UserIDBySession(context.Background(), sid)
	if err != nil || got != 0 {
		t.Fatalf("expired session should return 0, got %d %v", got, err)
	}
}
