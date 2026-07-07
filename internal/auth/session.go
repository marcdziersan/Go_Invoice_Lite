package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

const sessionCookieName = "go_invoice_session"

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]session
}

type session struct {
	UserID    int
	ExpiresAt time.Time
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]session)}
}

func (s *SessionStore) Create(w http.ResponseWriter, userID int) error {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}

	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	expires := time.Now().Add(8 * time.Hour)

	s.mu.Lock()
	s.sessions[token] = session{UserID: userID, ExpiresAt: expires}
	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *SessionStore) UserID(r *http.Request) (int, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return 0, false
	}

	s.mu.RLock()
	item, ok := s.sessions[cookie.Value]
	s.mu.RUnlock()
	if !ok {
		return 0, false
	}
	if time.Now().After(item.ExpiresAt) {
		s.DeleteToken(cookie.Value)
		return 0, false
	}
	return item.UserID, true
}

func (s *SessionStore) Destroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		s.DeleteToken(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *SessionStore) DeleteToken(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}
