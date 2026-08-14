// Package handler provides HTTP handlers (request/response only).
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/templeofair/sonix/internal/auth"
	"github.com/templeofair/sonix/internal/service"
)

// Auth handles /login, /logout, /me and auth middleware.
type Auth struct {
	svc     *service.AuthService
	mac     *auth.CookieMAC
	limiter *loginLimiter
}

// NewAuth returns an Auth handler. mac signs the session cookie.
func NewAuth(svc *service.AuthService, mac *auth.CookieMAC) *Auth {
	return &Auth{svc: svc, mac: mac, limiter: newLoginLimiter()}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type meResponse struct {
	Username string `json:"username"`
}

// Login handles POST /login. Same JSON and cookie behaviour as before.
func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}
	if !h.limiter.allow(clientIP(r.RemoteAddr)) {
		http.Error(w, "too many login attempts", http.StatusTooManyRequests)
		return
	}

	result, err := h.svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		log.Printf("auth: login failed: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    h.mac.Sign(result.SessionID),
		Path:     "/",
		MaxAge:   result.MaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   cookieSecure(r),
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

// Logout handles POST /logout.
func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var sessionID string
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		if id, ok := h.mac.Parse(cookie.Value); ok {
			sessionID = id
		}
	}
	_ = h.svc.Logout(r.Context(), sessionID)

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   cookieSecure(r),
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

// Me handles GET /me (must be wrapped with RequireAuth).
func (h *Auth) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(int64)
	if !ok || userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	username, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrUnauthorized) {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meResponse{Username: username})
}

// RequireAuth ensures a valid session cookie and sets auth.UserIDKey on the context.
func (h *Auth) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.CookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		sessionID, ok := h.mac.Parse(cookie.Value)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := h.svc.UserIDFromSession(r.Context(), sessionID)
		if err != nil || userID == 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func cookieSecure(r *http.Request) bool {
	return auth.SecureCookie || r.TLS != nil
}
