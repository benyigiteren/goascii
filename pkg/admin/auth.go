package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]string    // token -> username
	expiry   map[string]time.Time // token -> expiry time
}

var (
	GlobalSessionManager *SessionManager
	sessionOnce          sync.Once
)

func GetSessionManager() *SessionManager {
	sessionOnce.Do(func() {
		GlobalSessionManager = &SessionManager{
			sessions: make(map[string]string),
			expiry:   make(map[string]time.Time),
		}
		// Start a cleaner goroutine to remove expired sessions
		go GlobalSessionManager.startCleaner(5 * time.Minute)
	})
	return GlobalSessionManager
}

func (sm *SessionManager) CreateSession(username string, w http.ResponseWriter) string {
	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	// Set session to expire in 2 hours
	expires := time.Now().Add(2 * time.Hour)

	sm.mu.Lock()
	sm.sessions[token] = username
	sm.expiry[token] = expires
	sm.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  expires,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return token
}

func (sm *SessionManager) GetUsername(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return "", false
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	username, ok := sm.sessions[cookie.Value]
	if !ok {
		return "", false
	}

	exp, ok := sm.expiry[cookie.Value]
	if !ok || time.Now().After(exp) {
		return "", false
	}

	return username, true
}

func (sm *SessionManager) DestroySession(r *http.Request, w http.ResponseWriter) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sm.mu.Lock()
		delete(sm.sessions, cookie.Value)
		delete(sm.expiry, cookie.Value)
		sm.mu.Unlock()
	}

	// Expire the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		Path:     "/",
		HttpOnly: true,
	})
}

func (sm *SessionManager) startCleaner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for token, exp := range sm.expiry {
			if now.After(exp) {
				delete(sm.sessions, token)
				delete(sm.expiry, token)
			}
		}
		sm.mu.Unlock()
	}
}

// Password hashing utility

// HashPassword generates a random salt and hashes the password with it iteratively
func HashPassword(password string) (string, string, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", err
	}
	salt := hex.EncodeToString(saltBytes)
	hash := HashWithSalt(password, salt)
	return hash, salt, nil
}

// HashWithSalt performs 5000 iterations of SHA-256 with the password and salt
func HashWithSalt(password string, salt string) string {
	hashBytes := []byte(password + salt)
	for i := 0; i < 5000; i++ {
		h := sha256.Sum256(append(hashBytes, []byte(salt)...))
		hashBytes = h[:]
	}
	return hex.EncodeToString(hashBytes)
}

// VerifyPassword checks if the password matches the stored hash
func VerifyPassword(password, hash, salt string) bool {
	return HashWithSalt(password, salt) == hash
}
