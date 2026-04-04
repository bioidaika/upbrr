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

type tlHandler struct {
	cfg  config.Config
	http *http.Client
}

func (h tlHandler) Search(ctx context.Context, meta api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	_, _, ok := trackerCfgWithPasskey(h.cfg, "TL")
	if !ok {
		return nil, []string{noteSkip("missing passkey for tracker")}, nil
	}
	query := strings.TrimSpace(meta.Release.Title)
	if query == "" {
		query = strings.TrimSpace(meta.ReleaseName)
	}
	if query == "" {
		return nil, []string{noteSkip("missing title for TL dupe search")}, nil
	}
	endpoint := "https://www.torrentleech.org/torrents/browse/list/query/" + url.PathEscape(query)
	headers := map[string]string{"Accept": "application/json"}
	status, payload, err := doJSONGet(ctx, h.http, endpoint, nil, headers)
	if err != nil || status < 200 || status >= 300 || len(payload) == 0 {
		return nil, []string{noteSkip("TL search failed")}, nil
	}
	torrents, _ := payload["torrentList"].([]any)
	entries := make([]api.DupeEntry, 0, len(torrents))
	for _, raw := range torrents {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		fid := stringFromAny(item["fid"])
		entry := api.DupeEntry{Name: stringFromAny(item["name"]), ID: fid, Link: "https://www.torrentleech.org/torrent/" + fid}
		size := intFromAny(item["size"])
		if size > 0 {
			entry.SizeKnown = true
			entry.SizeBytes = size
		}
		entries = append(entries, entry)
	}
	return entries, nil, nil
}
