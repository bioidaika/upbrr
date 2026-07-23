// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package trackerauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/cookies"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestNETHDAuthCapabilityIsCookieOnly(t *testing.T) {
	t.Parallel()

	caps, err := NewService(config.Config{}).Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	for _, capability := range caps {
		if capability.TrackerID != "NETHD" {
			continue
		}
		if !capability.SupportsCookieFile || capability.SupportsLogin || capability.SupportsAutoLogin || capability.SupportsTOTP || capability.SupportsManual2FA {
			t.Fatalf("unexpected NETHD capability: %#v", capability)
		}
		return
	}
	t.Fatal("NETHD capability not found")
}

func TestNETHDCookieImportStillRequiresAnnounceURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := newTrackerAuthTestDB(t)
	service := NewService(config.Config{
		MainSettings: config.MainSettingsConfig{DBPath: dbPath},
		Trackers: config.TrackersConfig{Trackers: map[string]config.TrackerConfig{
			"NETHD": {},
		}},
	})
	status, err := service.ImportCookies(ctx, "NETHD", "cookies.json", `{"session":"fixture-cookie"}`)
	if err != nil {
		t.Fatalf("ImportCookies: %v", err)
	}
	if status.State != StateLoginRequired || status.CookieCount != 1 || !strings.Contains(status.Message, "announce_url") {
		t.Fatalf("expected announce prerequisite after cookie import, got %#v", status)
	}
	if IsReadyStatus(status) {
		t.Fatalf("NETHD must not be ready without announce_url: %#v", status)
	}

	service = NewService(config.Config{
		MainSettings: config.MainSettingsConfig{DBPath: dbPath},
		Trackers: config.TrackersConfig{Trackers: map[string]config.TrackerConfig{
			"NETHD": {AnnounceURL: "https://nethd.org/announce.php?passkey=fixture-passkey"},
		}},
	})
	status, err = service.Status(ctx, "NETHD")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.State != StateHasCookies || status.CookieCount != 1 || !IsReadyStatus(status) {
		t.Fatalf("expected local cookie readiness with announce_url, got %#v", status)
	}
}

func TestNETHDAnnounceURLBlocker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		announceURL string
		wantBlocker string
	}{
		{name: "missing", wantBlocker: nethdAnnounceMissingMessage},
		{name: "relative", announceURL: "/announce.php?passkey=fixture-passkey", wantBlocker: nethdAnnounceInvalidMessage},
		{name: "unsupported scheme", announceURL: "ftp://nethd.org/announce.php?passkey=fixture-passkey", wantBlocker: nethdAnnounceInvalidMessage},
		{name: "userinfo", announceURL: "https://user@nethd.org/announce.php?passkey=fixture-passkey", wantBlocker: nethdAnnounceInvalidMessage},
		{name: "missing passkey", announceURL: "https://nethd.org/announce.php", wantBlocker: nethdAnnouncePasskeyMessage},
		{name: "blank passkey", announceURL: "https://nethd.org/announce.php?passkey=%20", wantBlocker: nethdAnnouncePasskeyMessage},
		{name: "angle placeholder", announceURL: "https://nethd.org/announce.php?passkey=%3CPASSKEY%3E", wantBlocker: nethdAnnouncePlaceholderMessage},
		{name: "named placeholder", announceURL: "https://nethd.org/announce.php?passkey=your_passkey", wantBlocker: nethdAnnouncePlaceholderMessage},
		{name: "duplicate blank value", announceURL: "https://nethd.org/announce.php?passkey=fixture-passkey&PASSKEY=", wantBlocker: nethdAnnouncePasskeyMessage},
		{name: "valid HTTPS", announceURL: "https://nethd.org/announce.php?passkey=fixture-passkey"},
		{name: "valid HTTP and case-insensitive key", announceURL: "http://nethd.org/announce.php?PaSsKeY=fixture-passkey"},
		{name: "case-insensitive scheme", announceURL: "HTTPS://nethd.org/announce.php?passkey=fixture-passkey"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := nethdAnnounceURLBlocker(tt.announceURL); got != tt.wantBlocker {
				t.Fatalf("nethdAnnounceURLBlocker()=%q, want %q", got, tt.wantBlocker)
			}
		})
	}
}

func TestResolveNETHDStoredSessionRequiresAnnounceBeforeRemoteRequest(t *testing.T) {
	t.Parallel()

	requested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requested = true
		_, _ = w.Write([]byte(`<a href="/logout.php">Logout</a>`))
	}))
	defer server.Close()

	err := resolveNETHDStoredSessionForTrackerAuth(
		context.Background(),
		config.TrackerConfig{URL: server.URL},
		newTrackerAuthTestDB(t),
		api.TrackerAuthLoginRequest{},
	)
	var authRequired *AuthRequiredError
	if !errors.As(err, &authRequired) || !strings.Contains(authRequired.Reason, "announce_url") {
		t.Fatalf("expected missing announce_url AuthRequiredError, got %v", err)
	}
	if requested {
		t.Fatal("missing announce_url must fail before a remote request")
	}
}

func TestValidateNETHDPreservesSpecificAnnounceBlocker(t *testing.T) {
	t.Parallel()

	requested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requested = true
		_, _ = w.Write([]byte(`<a href="/logout.php">Logout</a>`))
	}))
	defer server.Close()

	ctx := context.Background()
	dbPath := newTrackerAuthTestDB(t)
	if err := cookies.SaveTrackerCookieMap(ctx, dbPath, "NETHD", map[string]string{"session": "fixture-cookie"}); err != nil {
		t.Fatalf("SaveTrackerCookieMap: %v", err)
	}
	status, err := NewService(config.Config{
		MainSettings: config.MainSettingsConfig{DBPath: dbPath},
		Trackers: config.TrackersConfig{Trackers: map[string]config.TrackerConfig{
			"NETHD": {URL: server.URL, AnnounceURL: "https://nethd.org/announce.php"},
		}},
	}).Validate(ctx, "NETHD")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if requested {
		t.Fatal("invalid announce_url must fail before a remote request")
	}
	if status.State != StateLoginRequired || status.CookieCount != 1 || status.Message != nethdAnnouncePasskeyMessage {
		t.Fatalf("expected specific announce_url blocker, got %#v", status)
	}
}

func TestResolveNETHDStoredSessionLoadsEncryptedAndLegacyCookies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cookieName string
		cookieVal  string
		seed       func(t *testing.T, dbPath string, domain string)
	}{
		{
			name:       "encrypted database",
			cookieName: "session",
			cookieVal:  "encrypted-fixture",
			seed: func(t *testing.T, dbPath string, _ string) {
				if err := cookies.SaveTrackerCookieMap(context.Background(), dbPath, "NETHD", map[string]string{"session": "encrypted-fixture"}); err != nil {
					t.Fatalf("SaveTrackerCookieMap: %v", err)
				}
			},
		},
		{
			name:       "legacy Netscape file",
			cookieName: "legacy_session",
			cookieVal:  "legacy-fixture",
			seed: func(t *testing.T, dbPath string, domain string) {
				cookiePath := trackerAuthLegacyCookiePathByExt(t, dbPath, "NETHD", ".txt")
				if err := os.MkdirAll(filepath.Dir(cookiePath), 0o700); err != nil {
					t.Fatalf("create cookie dir: %v", err)
				}
				payload := "#HttpOnly_" + domain + "\tFALSE\t/\tFALSE\t0\tlegacy_session\tlegacy-fixture\n"
				if err := os.WriteFile(cookiePath, []byte(payload), 0o600); err != nil {
					t.Fatalf("write legacy cookie: %v", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != nethdUploadPath {
					http.NotFound(w, r)
					return
				}
				cookie, err := r.Cookie(tt.cookieName)
				if err != nil || cookie.Value != tt.cookieVal {
					http.Error(w, "missing fixture cookie", http.StatusUnauthorized)
					return
				}
				_, _ = w.Write([]byte(`<a href="logout.php">Logout</a>`))
			}))
			defer server.Close()
			parsedServerURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("parse server URL: %v", err)
			}
			dbPath := newTrackerAuthTestDB(t)
			tt.seed(t, dbPath, parsedServerURL.Hostname())

			err = resolveNETHDStoredSessionForTrackerAuth(
				context.Background(),
				config.TrackerConfig{URL: server.URL, AnnounceURL: "https://nethd.org/announce.php?passkey=fixture-passkey"},
				dbPath,
				api.TrackerAuthLoginRequest{},
			)
			if err != nil {
				t.Fatalf("resolveNETHDStoredSessionForTrackerAuth: %v", err)
			}
		})
	}
}

func TestValidateNETHDStoredCookiesClassifiesRemoteResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		status           int
		location         string
		body             string
		wantInvalid      bool
		wantTransient    bool
		wantNoValidation bool
	}{
		{name: "logged in", status: http.StatusOK, body: `<html><a href="/logout.php">Logout</a></html>`, wantNoValidation: true},
		{name: "login redirect", status: http.StatusFound, location: "/login.php", wantInvalid: true},
		{name: "login form", status: http.StatusOK, body: `<form action="takelogin.php"><input name="username"><input name="password"></form>`, wantInvalid: true},
		{name: "missing marker", status: http.StatusOK, body: `<html>Example maintenance notice</html>`, wantTransient: true},
		{name: "server failure", status: http.StatusServiceUnavailable, body: `<input name="username">`, wantTransient: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.location != "" {
					w.Header().Set("Location", tt.location)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			err := validateNETHDStoredCookies(context.Background(), server.URL, []*http.Cookie{{Name: "session", Value: "fixture-cookie"}})
			if tt.wantNoValidation {
				if err != nil {
					t.Fatalf("expected valid session, got %v", err)
				}
				return
			}
			validationErr, ok := asValidationError(err)
			if !ok {
				t.Fatalf("expected ValidationError, got %v", err)
			}
			if validationErr.ConfirmedInvalid != tt.wantInvalid || validationErr.Transient != tt.wantTransient {
				t.Fatalf("unexpected classification: %+v", validationErr)
			}
		})
	}
}

func TestValidateNETHDDeletesOnlyConfirmedInvalidCookies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		status          int
		body            string
		wantCookieCount int
	}{
		{name: "confirmed invalid", status: http.StatusOK, body: `<form action="takelogin.php"><input name="username"></form>`},
		{name: "transient failure", status: http.StatusServiceUnavailable, body: "temporarily unavailable", wantCookieCount: 1},
		{name: "missing logout marker", status: http.StatusOK, body: "Example maintenance notice", wantCookieCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			dbPath := newTrackerAuthTestDB(t)
			if err := cookies.SaveTrackerCookieMap(ctx, dbPath, "NETHD", map[string]string{"session": "fixture-cookie"}); err != nil {
				t.Fatalf("SaveTrackerCookieMap: %v", err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			service := NewService(config.Config{
				MainSettings: config.MainSettingsConfig{DBPath: dbPath},
				Trackers: config.TrackersConfig{Trackers: map[string]config.TrackerConfig{
					"NETHD": {URL: server.URL, AnnounceURL: "https://nethd.org/announce.php?passkey=fixture-passkey"},
				}},
			})
			status, err := service.Validate(ctx, "NETHD")
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if status.CookieCount != tt.wantCookieCount {
				t.Fatalf("cookie count=%d, want %d: %#v", status.CookieCount, tt.wantCookieCount, status)
			}
			stored, loadErr := cookies.LoadTrackerCookieMap(ctx, dbPath, "NETHD")
			if tt.wantCookieCount == 0 {
				if loadErr == nil || len(stored) != 0 {
					t.Fatalf("confirmed-invalid cookies were not deleted: count=%d err=%v", len(stored), loadErr)
				}
				return
			}
			if loadErr != nil || len(stored) != 1 {
				t.Fatalf("transient failure must preserve cookies: count=%d err=%v", len(stored), loadErr)
			}
		})
	}
}
