package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AuthData struct {
	PasswordHash string `json:"password_hash"`
	Salt         string `json:"salt"`
}

type AuthManager struct {
	dataDir  string
	sessions map[string]time.Time
	mu       sync.RWMutex
}

func NewAuthManager(dataDir string) *AuthManager {
	return &AuthManager{
		dataDir:  dataDir,
		sessions: make(map[string]time.Time),
	}
}

func (am *AuthManager) authFilePath() string {
	return filepath.Join(am.dataDir, "auth.json")
}

func (am *AuthManager) IsSetup() bool {
	_, err := os.Stat(am.authFilePath())
	return err == nil
}

func (am *AuthManager) Setup(password string) error {
	if am.IsSetup() {
		return fmt.Errorf("password already configured")
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}

	saltHex := hex.EncodeToString(salt)
	hash := hashPassword(password, saltHex)

	data := AuthData{
		PasswordHash: hash,
		Salt:         saltHex,
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(am.authFilePath(), b, 0600)
}

func (am *AuthManager) Verify(password string) bool {
	b, err := os.ReadFile(am.authFilePath())
	if err != nil {
		return false
	}

	var data AuthData
	if err := json.Unmarshal(b, &data); err != nil {
		return false
	}

	return hashPassword(password, data.Salt) == data.PasswordHash
}

func (am *AuthManager) CreateSession() string {
	token := make([]byte, 32)
	rand.Read(token)
	tokenStr := hex.EncodeToString(token)

	am.mu.Lock()
	defer am.mu.Unlock()
	am.sessions[tokenStr] = time.Now().Add(24 * time.Hour)

	return tokenStr
}

func (am *AuthManager) ValidateSession(token string) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	expiry, ok := am.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(am.sessions, token)
		return false
	}
	return true
}

func (am *AuthManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Auth-Token")
		if token == "" {
			// Also check cookie
			if cookie, err := r.Cookie("auth_token"); err == nil {
				token = cookie.Value
			}
		}

		if !am.ValidateSession(token) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func hashPassword(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt + password))
	return hex.EncodeToString(h.Sum(nil))
}
