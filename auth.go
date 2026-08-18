package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookie = "sd_session"
	sessionTTL    = 7 * 24 * time.Hour
)

var sessions = struct {
	sync.Mutex
	m map[string]time.Time
}{m: make(map[string]time.Time)}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func setSession(w http.ResponseWriter) {
	token := newToken()
	sessions.Lock()
	sessions.m[token] = time.Now().Add(sessionTTL)
	sessions.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func isAuthenticated(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	sessions.Lock()
	defer sessions.Unlock()
	exp, ok := sessions.m[c.Value]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(sessions.m, c.Value)
		return false
	}
	return true
}

func clearSession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		sessions.Lock()
		delete(sessions.m, c.Value)
		sessions.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if isAuthenticated(r) {
		return true
	}
	writeErr(w, http.StatusUnauthorized, "未登录")
	return false
}

func hashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func checkPassword(pw string) bool {
	var hash string
	if err := db.QueryRow("SELECT value FROM settings WHERE key = 'password_hash'").Scan(&hash); err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
