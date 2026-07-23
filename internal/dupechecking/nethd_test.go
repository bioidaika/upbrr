// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestNETHDSearchUsesIMDbAndParsesNexusLinks(t *testing.T) {
	t.Parallel()

	queryCh := make(chan url.Values, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/torrents.php" {
			http.NotFound(w, r)
			return
		}
		if cookie, err := r.Cookie("session"); err != nil || cookie.Value != "cookie" {
			http.Error(w, "fixture cookie missing", http.StatusUnauthorized)
			return
		}
		queryCh <- r.URL.Query()
		_, _ = w.Write([]byte(`<html><body><table class="torrent">
			<tr><td><a href="/details.php?id=41&amp;hit=1" title="Example's.Release.2026.1080p.WEB-DL-GRP">Example</a></td><td>1.50 GiB</td></tr>
			<tr><td><a href="/Example-Release-2026-2160p-WEB-DL-GRP-torrent-42.html">Example Release</a></td><td>12 GB</td></tr>
			<tr><td><a href="/torrent-43-Example-Release-2026-720p-GRP.html">Short</a></td><td>900 MiB</td></tr>
			<tr><td><a href="/forums/torrent-rules.html">Ignore me</a></td><td>4 GiB</td></tr>
		</table></body></html>`))
	}))
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "upbrr.db")
	writeTextCookie(t, dbPath, "NETHD", hostFromBaseURL(t, server.URL))
	cfg := config.Config{
		MainSettings: config.MainSettingsConfig{DBPath: dbPath},
		Trackers: config.TrackersConfig{Trackers: map[string]config.TrackerConfig{
			"NETHD": {URL: server.URL},
		}},
	}
	entries, notes, err := (nethdHandler{cfg: cfg, http: server.Client()}).Search(
		context.Background(),
		api.PreparedMetadata{ExternalIDs: api.ExternalIDs{IMDBID: 1234567}},
		"NETHD",
	)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("unexpected notes: %v", notes)
	}
	query := <-queryCh
	for key, want := range map[string]string{
		"search":      "tt1234567",
		"search_area": "4",
		"search_mode": "0",
		"incldead":    "0",
	} {
		if got := query.Get(key); got != want {
			t.Fatalf("query %s=%q, want %q", key, got, want)
		}
	}
	if len(entries) != 3 {
		t.Fatalf("expected three NETHD entries, got %#v", entries)
	}
	if entries[0].ID != "41" || entries[0].Name != "Example's.Release.2026.1080p.WEB-DL-GRP" || !entries[0].SizeKnown || entries[0].SizeText != "1.50 GiB" {
		t.Fatalf("unexpected standard details entry: %#v", entries[0])
	}
	if entries[1].ID != "42" || entries[1].Name != "Example Release 2026 2160p WEB DL GRP" || entries[1].SizeBytes != 12_000_000_000 {
		t.Fatalf("unexpected name-first SEO entry: %#v", entries[1])
	}
	if entries[2].ID != "43" || entries[2].Name != "Example Release 2026 720p GRP" || !strings.HasSuffix(entries[2].Link, "/torrent-43-Example-Release-2026-720p-GRP.html") {
		t.Fatalf("unexpected ID-first SEO entry: %#v", entries[2])
	}
}

func TestNETHDSearchFallsBackToTitle(t *testing.T) {
	t.Parallel()

	queryCh := make(chan url.Values, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryCh <- r.URL.Query()
		_, _ = w.Write([]byte(`<table class="torrents"></table>`))
	}))
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "upbrr.db")
	writeTextCookie(t, dbPath, "NETHD", hostFromBaseURL(t, server.URL))
	cfg := config.Config{
		MainSettings: config.MainSettingsConfig{DBPath: dbPath},
		Trackers: config.TrackersConfig{Trackers: map[string]config.TrackerConfig{
			"NETHD": {URL: server.URL},
		}},
	}
	entries, notes, err := (nethdHandler{cfg: cfg, http: server.Client()}).Search(
		context.Background(),
		api.PreparedMetadata{Release: api.ReleaseInfo{Title: "Example Release"}},
		"NETHD",
	)
	if err != nil || len(notes) != 0 || len(entries) != 0 {
		t.Fatalf("unexpected title fallback result: entries=%#v notes=%v err=%v", entries, notes, err)
	}
	query := <-queryCh
	if query.Get("search") != "Example Release" || query.Get("search_area") != "0" || query.Get("search_mode") != "0" || query.Get("incldead") != "0" {
		t.Fatalf("unexpected title fallback query: %v", query)
	}
}

func TestParseNETHDDetailHref(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		href     string
		wantID   string
		wantSlug string
		wantOK   bool
	}{
		{name: "standard", href: "details.php?id=123&uploaded=1", wantID: "123", wantOK: true},
		{name: "standard absolute", href: "https://nethd.org/details.php?foo=bar&id=124", wantID: "124", wantOK: true},
		{name: "name first SEO", href: "/Example-Release-1080p-torrent-125.html", wantID: "125", wantSlug: "Example-Release-1080p", wantOK: true},
		{name: "ID first SEO", href: "/torrent-126-Example-Release-720p.html", wantID: "126", wantSlug: "Example-Release-720p", wantOK: true},
		{name: "missing ID", href: "/details.php", wantOK: false},
		{name: "non decimal ID", href: "/details.php?id=abc", wantOK: false},
		{name: "unrelated torrent link", href: "/forums/torrent-rules.html", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, slug, ok := parseNETHDDetailHref(tt.href)
			if ok != tt.wantOK || id != tt.wantID || slug != tt.wantSlug {
				t.Fatalf("parseNETHDDetailHref(%q)=(%q,%q,%t), want (%q,%q,%t)", tt.href, id, slug, ok, tt.wantID, tt.wantSlug, tt.wantOK)
			}
		})
	}
}

func TestBuildHandlersRegistersNETHD(t *testing.T) {
	t.Parallel()

	handler, ok := buildHandlers(handlerDeps{cfg: config.Config{}, http: http.DefaultClient})["NETHD"]
	if !ok {
		t.Fatal("NETHD dupe handler not registered")
	}
	if _, ok := handler.(nethdHandler); !ok {
		t.Fatalf("unexpected NETHD handler type %T", handler)
	}
}
