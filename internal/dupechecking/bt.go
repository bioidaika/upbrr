// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/pkg/api"
)

type btHandler struct {
	cfg  config.Config
	http *http.Client
}

func (h btHandler) Search(ctx context.Context, meta api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	cookies, err := loadTrackerTextCookies(h.cfg, "BT", trackerHost(trackerBaseURL(h.cfg, "BT", "https://brasiltracker.org"), "brasiltracker.org"))
	if err != nil {
		return nil, []string{noteSkip("missing valid BT cookies")}, nil
	}
	query := strings.TrimSpace(meta.Release.Title)
	if strings.EqualFold(categoryOfSiteMeta(meta), "TV") {
		query = strings.TrimSpace(strings.TrimSpace(query) + " " + resolveSeasonEpisodeQuery(meta))
	}
	if query == "" {
		query = strings.TrimSpace(meta.ReleaseName)
	}
	if query == "" {
		return nil, []string{noteSkip("missing title for BT dupe search")}, nil
	}
	baseURL := trackerBaseURL(h.cfg, "BT", "https://brasiltracker.org")
	resp, body, err := doTextGet(ctx, h.http, baseURL+"/suggest.php", url.Values{"q": {query}}, nil, cookies)
	if err != nil || resp == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, []string{noteSkip("BT search failed")}, nil
	}
	if resp.Request != nil && resp.Request.URL != nil && strings.Contains(strings.ToLower(resp.Request.URL.String()), "login") {
		return nil, []string{noteSkip("BT authentication required")}, nil
	}
	lines := strings.Split(body, "\n")
	entries := make([]api.DupeEntry, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			entries = append(entries, api.DupeEntry{Name: trimmed})
		}
	}
	return entries, nil, nil
}
