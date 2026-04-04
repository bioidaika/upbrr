// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package webserver

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	sessionCookieName = "ua_web_session"
	authFileName      = "web-auth.json"
)

type authRecord struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type authStore struct {
	path string
	mu   sync.Mutex
}

func newAuthStore(dbPath string) (*authStore, error) {
	dir := filepath.Dir(strings.TrimSpace(dbPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("web auth: create config dir: %w", err)
	}
	return &authStore{path: filepath.Join(dir, authFileName)}, nil
}

func (s *authStore) Exists() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := os.Stat(s.path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *authStore) Load() (authRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(s.path)
	if err != nil {
		return authRecord{}, err
	}
	var record authRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return authRecord{}, err
	}
	return record, nil
}

func (s *authStore) Bootstrap(username string, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.path); err == nil {
		return errors.New("web auth: user already exists")
	}

	hash, err := hashPassword(password)
	if err != nil {
		return err
	}

	record := authRecord{
		Username:     strings.TrimSpace(username),
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func hashPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if len(password) < 10 {
		return "", errors.New("password must be at least 10 characters")
	}
	salt, err := randomString(16)
	if err != nil {
		return "", err
	}
	sum := argon2.IDKey([]byte(password), []byte(salt), 1, 64*1024, 4, 32)
	return fmt.Sprintf("argon2id$%s$%s", salt, base64.RawStdEncoding.EncodeToString(sum)), nil
}

func verifyPassword(password string, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 3 || parts[0] != "argon2id" {
		return false
	}
	sum := argon2.IDKey([]byte(password), []byte(parts[1]), 1, 64*1024, 4, 32)
	expected, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(sum, expected) == 1
}

type session struct {
	ID        string
	Username  string
	CSRFToken string
	ExpiresAt time.Time
}

type sessionManager struct {
	ttl          time.Duration
	cleanupEvery time.Duration
	stopCh       chan struct{}
	doneCh       chan struct{}
	stopOnce     sync.Once
	mu           sync.Mutex
	sessions     map[string]session
}

func newSessionManager(ttlMinutes int) *sessionManager {
	if ttlMinutes <= 0 {
		ttlMinutes = 1440
	}
	manager := &sessionManager{
		ttl:          time.Duration(ttlMinutes) * time.Minute,
		cleanupEvery: time.Minute,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
		sessions:     make(map[string]session),
	}
	go manager.cleanupLoop()
	return manager
}

func (m *sessionManager) Create(username string) (session, error) {
	id, err := randomString(32)
	if err != nil {
		return session{}, err
	}
	csrf, err := randomString(24)
	if err != nil {
		return session{}, err
	}

	current := session{
		ID:        id,
		Username:  strings.TrimSpace(username),
		CSRFToken: csrf,
		ExpiresAt: time.Now().UTC().Add(m.ttl),
	}

	m.mu.Lock()
	m.sessions[id] = current
	m.mu.Unlock()

	return current, nil
}

func (m *sessionManager) Get(id string) (session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.sessions[id]
	if !ok {
		return session{}, false
	}
	if time.Now().UTC().After(current.ExpiresAt) {
		delete(m.sessions, id)
		return session{}, false
	}
	return current, true
}

func (m *sessionManager) Delete(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

func (m *sessionManager) Close() {
	if m == nil {
		return
	}
	m.stopOnce.Do(func() {
		close(m.stopCh)
		<-m.doneCh
	})
}

func (m *sessionManager) cleanupLoop() {
	ticker := time.NewTicker(m.cleanupEvery)
	defer func() {
		ticker.Stop()
		close(m.doneCh)
	}()

	for {
		select {
		case <-ticker.C:
			m.deleteExpired(time.Now().UTC())
		case <-m.stopCh:
			return
		}
	}
}

func (m *sessionManager) deleteExpired(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, current := range m.sessions {
		if now.After(current.ExpiresAt) {
			delete(m.sessions, id)
		}
	}
}

func randomString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func ipFromAddr(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		return strings.TrimSpace(remoteAddr)
	}
	return host
}

type rateLimitBucket struct {
	Count   int
	ResetAt time.Time
}

type fixedWindowLimiter struct {
	limit  int
	window time.Duration
	mu     sync.Mutex
	items  map[string]rateLimitBucket
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		limit:  limit,
		window: window,
		items:  make(map[string]rateLimitBucket),
	}
}

func (l *fixedWindowLimiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	now := time.Now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.items[key]
	if now.After(current.ResetAt) {
		current = rateLimitBucket{ResetAt: now.Add(l.window)}
	}
	if current.Count >= l.limit {
		l.items[key] = current
		return false
	}
	current.Count++
	l.items[key] = current
	return true
}
