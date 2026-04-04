// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestSceneDetectorSRRDB(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/search/r:Example.Release", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resultsCount":1,"results":[{"release":"Example.Release.2024.1080p-WEB","imdbId":"1234567"}]}`))
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cacheDir := t.TempDir()
	nfoDir := t.TempDir()
	detector := newSRRDBDetector(server.Client(), server.URL, cacheDir, nfoDir)

	meta := api.PreparedMetadata{VideoPath: "/data/Example.Release.mkv"}
	result, err := detector.Detect(context.Background(), meta)
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if !result.IsScene {
		t.Fatalf("expected scene match")
	}
	if !strings.HasPrefix(result.SceneName, "Example.Release") {
		t.Fatalf("unexpected scene name: %q", result.SceneName)
	}
	if result.IMDBID != 1234567 {
		t.Fatalf("unexpected imdb id: %d", result.IMDBID)
	}
}
