package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"

	runtimeauth "github.com/indu-forge/runtime-api/internal/auth"
)

const sessionCookieName = "induforge_runtime_session"

type session struct {
	principal runtimeauth.Principal
	expiresAt time.Time
}

type sessionManager struct {
	mu       sync.Mutex
	sessions map[string]session
	ttl      time.Duration
}

func newSessionManager() *sessionManager {
	return &sessionManager{sessions: map[string]session{}, ttl: 8 * time.Hour}
}

func (m *sessionManager) issue(principal runtimeauth.Principal, now time.Time) (string, time.Time, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", time.Time{}, err
	}
	id := base64.RawURLEncoding.EncodeToString(random[:])
	expiresAt := now.Add(m.ttl)
	if principal.ExpiresAt != nil && principal.ExpiresAt.Before(expiresAt) {
		expiresAt = principal.ExpiresAt.UTC()
	}
	if !now.Before(expiresAt) {
		return "", time.Time{}, errors.New("access token 已过期")
	}
	m.mu.Lock()
	m.removeExpiredLocked(now)
	m.sessions[id] = session{principal: principal, expiresAt: expiresAt}
	m.mu.Unlock()
	return id, expiresAt, nil
}

func (m *sessionManager) authenticate(id string, now time.Time) (runtimeauth.Principal, error) {
	if id == "" || len(id) > 128 {
		return runtimeauth.Principal{}, errors.New("invalid runtime session")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.sessions[id]
	if !ok || !now.Before(item.expiresAt) {
		delete(m.sessions, id)
		return runtimeauth.Principal{}, errors.New("invalid runtime session")
	}
	return item.principal, nil
}

func (m *sessionManager) revoke(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

func (m *sessionManager) removeExpiredLocked(now time.Time) {
	for id, item := range m.sessions {
		if !now.Before(item.expiresAt) {
			delete(m.sessions, id)
		}
	}
}

func sessionCookie(id string, now, expiresAt time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(expiresAt.Sub(now).Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}

func expiredSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}
