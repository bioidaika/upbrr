// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nethd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/anacrolix/torrent/metainfo"
	mkbrr "github.com/autobrr/mkbrr/torrent"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestBuildPayloadMatchesNETHDForm(t *testing.T) {
	meta := api.PreparedMetadata{
		ReleaseName:       " Example.Release.2026.2160p-GRP ",
		TVPack:            true,
		AudioLanguages:    []string{"Vietnamese"},
		SubtitleLanguages: []string{"vi-VN"},
		Type:              "WEB-DL",
		Release: api.ReleaseInfo{
			Category:   "TV",
			Year:       2026,
			Resolution: "2160p",
		},
		ExternalIDs: api.ExternalIDs{TMDBID: 123456, IMDBID: 1234567, Category: "TV"},
		ExternalMetadata: api.ExternalMetadata{TMDB: &api.TMDBMetadata{
			Year:   2026,
			Poster: "https://images.example/poster.jpg",
		}},
	}

	payload := buildPayload(meta, "description")
	want := map[string]string{
		"name":        "Example.Release.2026.2160p-GRP",
		"small_descr": "(2026) (TM/LT) (VietSub) tv/123456",
		"poster":      "https://images.example/poster.jpg",
		"type":        "401",
		"subcategory": "550",
		"source":      "410",
		"standard":    "419",
		"url":         "https://www.imdb.com/title/tt1234567/",
		"descr":       "description",
		"team_sel":    "0",
	}
	for key, expected := range want {
		if got := payload[key]; got != expected {
			t.Fatalf("payload[%q] = %q, want %q", key, got, expected)
		}
	}
}

func TestResolveSubcategoryIDPriorityAndTV(t *testing.T) {
	tests := []struct {
		name string
		meta api.PreparedMetadata
		want int
	}{
		{
			name: "animation wins over action",
			meta: api.PreparedMetadata{ExternalMetadata: api.ExternalMetadata{TMDB: &api.TMDBMetadata{Genres: "Action, Animation"}}},
			want: 425,
		},
		{
			name: "keywords supplement genres",
			meta: api.PreparedMetadata{ExternalMetadata: api.ExternalMetadata{TMDB: &api.TMDBMetadata{Genres: "Drama", Keywords: "science fiction"}}},
			want: 431,
		},
		{name: "other", meta: api.PreparedMetadata{}, want: 439},
		{name: "tv episode", meta: api.PreparedMetadata{SeasonInt: 1, EpisodeInt: 2}, want: 511},
		{name: "tv pack", meta: api.PreparedMetadata{TVPack: true}, want: 550},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolveSubcategoryID(test.meta); got != test.want {
				t.Fatalf("resolveSubcategoryID() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestResolveSourceAndStandardIDs(t *testing.T) {
	sourceTests := []struct {
		name string
		meta api.PreparedMetadata
		want int
	}{
		{name: "bluray disc", meta: api.PreparedMetadata{DiscType: "BDMV", Type: "ENCODE"}, want: 411},
		{name: "dvd disc", meta: api.PreparedMetadata{DiscType: "DVD"}, want: 414},
		{name: "remux", meta: api.PreparedMetadata{Type: "REMUX"}, want: 555},
		{name: "encode", meta: api.PreparedMetadata{Type: "ENCODE"}, want: 556},
		{name: "web dl", meta: api.PreparedMetadata{Type: "WEBDL"}, want: 410},
		{name: "web rip", meta: api.PreparedMetadata{Type: "WEBRIP"}, want: 556},
		{name: "hdtv", meta: api.PreparedMetadata{Type: "HDTV"}, want: 413},
		{name: "dvd rip", meta: api.PreparedMetadata{Type: "DVDRIP"}, want: 414},
		{name: "category type falls through", meta: api.PreparedMetadata{Type: "MOVIE", Release: api.ReleaseInfo{Type: "REMUX"}}, want: 555},
		{name: "other", meta: api.PreparedMetadata{}, want: 530},
	}
	for _, test := range sourceTests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolveSourceID(test.meta); got != test.want {
				t.Fatalf("resolveSourceID() = %d, want %d", got, test.want)
			}
		})
	}

	standardTests := map[string]int{
		"4320p": 557,
		"2160p": 419,
		"1080p": 415,
		"1080i": 415,
		"720p":  416,
		"576p":  418,
		"480i":  418,
		"":      418,
	}
	for resolution, expected := range standardTests {
		meta := api.PreparedMetadata{Release: api.ReleaseInfo{Resolution: resolution}}
		if got := resolveStandardID(meta); got != expected {
			t.Fatalf("resolveStandardID(%q) = %d, want %d", resolution, got, expected)
		}
	}
}

func TestBuildDescriptionPreservesFinalAssets(t *testing.T) {
	final := "[center]A | B\n\n[url=https://img.example/a.png][img=1000]https://img.example/a.png[/img][/url][/center]"
	got := buildDescription(trackers.UploadRequest{}, trackers.DescriptionAssets{
		Final:       true,
		Description: "\n" + final + "\n",
	})
	if got != final {
		t.Fatalf("final description changed: got %q want %q", got, final)
	}
}

func TestBuildDescriptionGatesTVOverviewAndIncludesEpisodeImage(t *testing.T) {
	req := trackers.UploadRequest{
		Meta: api.PreparedMetadata{
			Release:         api.ReleaseInfo{Category: "TV"},
			EpisodeTitle:    "Example Episode",
			EpisodeOverview: "Example overview.",
			ExternalMetadata: api.ExternalMetadata{TVDB: &api.TVDBMetadata{
				EpisodeImage: "https://img.example/episode.jpg",
			}},
		},
	}

	withoutOverview := buildDescription(req, trackers.DescriptionAssets{})
	if strings.Contains(withoutOverview, "Example Episode") || strings.Contains(withoutOverview, "Example overview.") || strings.Contains(withoutOverview, "episode.jpg") {
		t.Fatalf("TV overview ignored disabled setting: %q", withoutOverview)
	}

	req.AppConfig.Description.EpisodeOverview = true
	withOverview := buildDescription(req, trackers.DescriptionAssets{})
	for _, expected := range []string{
		"[center]Example Episode[/center]",
		"[center][img]https://img.example/episode.jpg[/img][/center]",
		"[center]Example overview.[/center]",
	} {
		if !strings.Contains(withOverview, expected) {
			t.Fatalf("TV description missing %q: %q", expected, withOverview)
		}
	}
}

func TestParseUploadResultAcceptsStandardAndSEOURLs(t *testing.T) {
	tests := []struct {
		name     string
		location string
		wantID   string
	}{
		{name: "standard", location: "/details.php?id=123&uploaded=1", wantID: "123"},
		{name: "slug first", location: "/example-release-torrent-456.html", wantID: "456"},
		{name: "torrent first", location: "/torrent-789-example-release.html", wantID: "789"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id, detailURL, err := parseUploadResult("https://nethd.org", test.location)
			if err != nil {
				t.Fatalf("parseUploadResult: %v", err)
			}
			if id != test.wantID {
				t.Fatalf("id = %q, want %q", id, test.wantID)
			}
			if strings.Contains(detailURL, "uploaded=") {
				t.Fatalf("detail URL retained upload marker: %s", detailURL)
			}
		})
	}

	if _, _, err := parseUploadResult("https://nethd.org", "https://outside.example/details.php?id=123"); err == nil {
		t.Fatal("expected cross-origin redirect rejection")
	}

	_, detailURL, err := parseUploadResult("https://nethd.org", "/details.php?id=123&uploaded=1&passkey=secret#private")
	if err != nil {
		t.Fatalf("parseUploadResult sensitive redirect: %v", err)
	}
	if detailURL != "https://nethd.org/details.php?id=123" {
		t.Fatalf("detail URL = %q, want canonical URL without sensitive values", detailURL)
	}
}

func TestValidateAnnounceURLRequiresPasskey(t *testing.T) {
	for _, value := range []string{
		"",
		"not-a-url",
		"https://:443/announce.php?passkey=fixture-passkey",
		"https://nethd.org/announce.php",
		"https://nethd.org/announce.php?passkey=%3CPASSKEY%3E",
		"https://nethd.org/announce.php?passkey=passkey",
		"https://nethd.org/announce.php?passkey=fixture-passkey&PASSKEY=",
	} {
		if err := validateAnnounceURL(value); err == nil {
			t.Fatalf("validateAnnounceURL(%q) unexpectedly succeeded", value)
		}
	}
	for _, value := range []string{
		"https://nethd.org/announce.php?PassKey=test-passkey",
		"HTTPS://nethd.org/announce.php?passkey=test-passkey",
	} {
		if err := validateAnnounceURL(value); err != nil {
			t.Fatalf("validateAnnounceURL(%q): %v", value, err)
		}
	}
}

func TestUploadPostsMultipartAndKeepsLocallyGeneratedTorrent(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "upbrr.db")
	mediaPath := filepath.Join(tmp, "Example.Release.2026.mkv")
	torrentPath := filepath.Join(tmp, "base.torrent")
	createNETHDTestTorrent(t, mediaPath, torrentPath)

	var (
		mu               sync.Mutex
		uploadedTorrent  []byte
		uploadFields     map[string]string
		downloadRequests int
	)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if cookie, err := request.Cookie("session"); err != nil || cookie.Value != "test-session" {
			http.Error(writer, "missing session", http.StatusUnauthorized)
			return
		}
		switch request.URL.Path {
		case "/takeupload.php":
			fields, torrentBytes, err := readNETHDUpload(request)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			uploadFields = fields
			uploadedTorrent = torrentBytes
			mu.Unlock()
			writer.Header().Set("Location", "/details.php?id=123&uploaded=1")
			writer.WriteHeader(http.StatusFound)
		case "/download.php":
			mu.Lock()
			downloadRequests++
			mu.Unlock()
			http.Error(writer, "unexpected torrent download", http.StatusInternalServerError)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	writeNETHDCookieFile(t, dbPath, serverURL.Hostname())

	req := trackers.UploadRequest{
		Tracker: trackerName,
		Meta: api.PreparedMetadata{
			SourcePath:  mediaPath,
			TorrentPath: torrentPath,
			ReleaseName: "Example.Release.2026.1080p-GRP",
			Type:        "WEBDL",
			Release:     api.ReleaseInfo{Category: "MOVIE", Year: 2026, Resolution: "1080p"},
			ExternalIDs: api.ExternalIDs{TMDBID: 123456, IMDBID: 1234567, Category: "MOVIE"},
		},
		TrackerConfig: config.TrackerConfig{
			URL:         server.URL,
			AnnounceURL: server.URL + "/announce.php?passkey=test-passkey",
		},
		AppConfig: config.Config{MainSettings: config.MainSettingsConfig{DBPath: dbPath}},
		Logger:    api.NopLogger{},
	}

	summary, err := upload(context.Background(), req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if summary.Uploaded != 1 || len(summary.UploadedTorrents) != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	artifact := summary.UploadedTorrents[0]
	if artifact.TorrentID != "123" || artifact.TorrentPath == "" {
		t.Fatalf("unexpected uploaded torrent: %#v", artifact)
	}
	expectedArtifactPath, err := trackers.ResolveTrackerTorrentArtifactPath(req.Meta, dbPath, trackerName)
	if err != nil {
		t.Fatalf("resolve tracker artifact path: %v", err)
	}
	if artifact.TorrentPath != expectedArtifactPath {
		t.Fatalf("TorrentPath = %q, want local artifact %q", artifact.TorrentPath, expectedArtifactPath)
	}

	mu.Lock()
	fields := uploadFields
	uploadedPayload := append([]byte(nil), uploadedTorrent...)
	downloadCount := downloadRequests
	mu.Unlock()
	if fields["type"] != "401" || fields["source"] != "410" || fields["standard"] != "415" {
		t.Fatalf("unexpected upload fields: %#v", fields)
	}
	if downloadCount != 0 {
		t.Fatalf("expected no /download.php request, got %d", downloadCount)
	}

	artifactPayload, err := os.ReadFile(artifact.TorrentPath)
	if err != nil {
		t.Fatalf("read local artifact: %v", err)
	}
	if !bytes.Equal(artifactPayload, uploadedPayload) {
		t.Fatal("local artifact bytes differ from the torrent uploaded to NETHD")
	}
	torrentMeta, err := metainfo.Load(bytes.NewReader(artifactPayload))
	if err != nil {
		t.Fatalf("parse local artifact: %v", err)
	}
	if torrentMeta.Announce != req.TrackerConfig.AnnounceURL {
		t.Fatalf("announce = %q, want %q", torrentMeta.Announce, req.TrackerConfig.AnnounceURL)
	}
	info, err := torrentMeta.UnmarshalInfo()
	if err != nil {
		t.Fatalf("unmarshal artifact info: %v", err)
	}
	if info.Source != sourceFlag {
		t.Fatalf("source = %q, want %q", info.Source, sourceFlag)
	}
}

func createNETHDTestTorrent(t *testing.T, sourcePath string, torrentPath string) {
	t.Helper()
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		if err := os.WriteFile(sourcePath, []byte("test media data"), 0o600); err != nil {
			t.Fatalf("write source: %v", err)
		}
	}
	_, err := mkbrr.Create(mkbrr.CreateOptions{
		Path:       sourcePath,
		OutputPath: torrentPath,
		IsPrivate:  true,
	})
	if err != nil {
		t.Fatalf("create torrent: %v", err)
	}
}

func writeNETHDCookieFile(t *testing.T, dbPath string, domain string) {
	t.Helper()
	cookieDir := filepath.Join(filepath.Dir(dbPath), "cookies")
	if err := os.MkdirAll(cookieDir, 0o700); err != nil {
		t.Fatalf("create cookie dir: %v", err)
	}
	content := "# Netscape HTTP Cookie File\n" + domain + "\tFALSE\t/\tFALSE\t0\tsession\ttest-session\n"
	if err := os.WriteFile(filepath.Join(cookieDir, trackerName+".txt"), []byte(content), 0o600); err != nil {
		t.Fatalf("write cookie file: %v", err)
	}
}

func readNETHDUpload(request *http.Request) (map[string]string, []byte, error) {
	reader, err := request.MultipartReader()
	if err != nil {
		return nil, nil, err
	}
	fields := map[string]string{}
	var torrentBytes []byte
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		payload, err := io.ReadAll(part)
		if err != nil {
			return nil, nil, err
		}
		if part.FileName() != "" {
			if part.FormName() == "file" {
				torrentBytes = payload
			}
			continue
		}
		fields[part.FormName()] = string(payload)
	}
	if len(torrentBytes) == 0 {
		return nil, nil, io.ErrUnexpectedEOF
	}
	return fields, torrentBytes, nil
}
