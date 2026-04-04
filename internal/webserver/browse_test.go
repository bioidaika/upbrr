// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package webserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubNativePicker struct {
	filePath    string
	folderPath  string
	fileErr     error
	folderErr   error
	fileCalls   int
	folderCalls int
}

func (s *stubNativePicker) BrowseFile() (string, error) {
	s.fileCalls++
	return s.filePath, s.fileErr
}

func (s *stubNativePicker) BrowseFolder() (string, error) {
	s.folderCalls++
	return s.folderPath, s.folderErr
}

func testSessionManager() *sessionManager {
	return &sessionManager{
		ttl:      time.Hour,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		sessions: map[string]session{},
	}
}

func testServerWithPicker(picker nativePicker) *Server {
	manager := testSessionManager()
	manager.sessions["test-session"] = session{
		ID:        "test-session",
		Username:  "tester",
		CSRFToken: "test-csrf",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	return &Server{
		picker:         picker,
		sessions:       manager,
		generalLimiter: newFixedWindowLimiter(100, time.Minute),
		authLimiter:    newFixedWindowLimiter(100, time.Minute),
	}
}

func newBrowseRequest(path string, host string, remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
	req.Host = host
	req.RemoteAddr = remoteAddr
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://"+host)
	req.Header.Set("X-CSRF-Token", "test-csrf")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	return req
}

func TestIsLoopbackHostname(t *testing.T) {
	t.Parallel()

	cases := []struct {
		host string
		want bool
	}{
		{host: "localhost", want: true},
		{host: "sub.localhost", want: true},
		{host: "127.0.0.1", want: true},
		{host: "::1", want: true},
		{host: "192.168.1.20", want: false},
		{host: "example.com", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			t.Parallel()
			if got := isLoopbackHostname(tc.host); got != tc.want {
				t.Fatalf("isLoopbackHostname(%q) = %v, want %v", tc.host, got, tc.want)
			}
		})
	}
}

func TestHandleAuthStatusIncludesNativeBrowseCapability(t *testing.T) {
	store, err := newAuthStore(filepath.Join(t.TempDir(), "state", "db.sqlite"))
	if err != nil {
		t.Fatalf("newAuthStore: %v", err)
	}
	server := &Server{
		auth:   store,
		picker: &stubNativePicker{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	req.Host = "127.0.0.1:8080"
	req.RemoteAddr = "127.0.0.1:5050"

	recorder := httptest.NewRecorder()
	server.handleAuthStatus(recorder, req, session{})

	if recorder.Code != http.StatusOK {
		t.Fatalf("handleAuthStatus returned %d", recorder.Code)
	}

	var payload struct {
		NativeBrowseEnabled bool `json:"nativeBrowseEnabled"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal auth status: %v", err)
	}
	if !payload.NativeBrowseEnabled {
		t.Fatal("expected localhost auth status to advertise native browse support")
	}
}

func TestBrowseFileRouteAllowsLocalhostSessions(t *testing.T) {
	picker := &stubNativePicker{filePath: `C:\Media\movie.mkv`}
	server := testServerWithPicker(picker)
	mux := http.NewServeMux()
	server.registerAppRoutes(mux)

	recorder := httptest.NewRecorder()
	req := newBrowseRequest("/api/app/BrowseFile", "127.0.0.1:8080", "127.0.0.1:5050")
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("browse file route returned %d", recorder.Code)
	}
	if picker.fileCalls != 1 {
		t.Fatalf("expected picker to be called once, got %d", picker.fileCalls)
	}
	if got := strings.TrimSpace(recorder.Body.String()); !strings.Contains(got, `C:\\Media\\movie.mkv`) {
		t.Fatalf("expected response to include selected path, got %q", got)
	}
}

func TestBrowseFileRouteRejectsRemoteSessions(t *testing.T) {
	picker := &stubNativePicker{filePath: `C:\Media\movie.mkv`}
	server := testServerWithPicker(picker)
	mux := http.NewServeMux()
	server.registerAppRoutes(mux)

	req := newBrowseRequest("/api/app/BrowseFile", "example.com:8080", "192.168.1.25:5050")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden status, got %d", recorder.Code)
	}
	if picker.fileCalls != 0 {
		t.Fatalf("expected picker not to be called, got %d calls", picker.fileCalls)
	}
	if !strings.Contains(recorder.Body.String(), "localhost web sessions") {
		t.Fatalf("expected remote browse error message, got %q", recorder.Body.String())
	}
}
