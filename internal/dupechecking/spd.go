// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/pkg/api"
)

type spdHandler struct {
	cfg  config.Config
	http *http.Client
}

func (h spdHandler) Search(ctx context.Context, meta api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	_, apiKey, ok := trackerCfgWithAPIKey(h.cfg, "SPD")
	if !ok {
		return nil, []string{noteSkip("missing api_key for tracker")}, nil
	}
	params := url.Values{}
	switch {
	case meta.ExternalIDs.IMDBID != 0:
		params.Set("imdbId", strconv.Itoa(meta.ExternalIDs.IMDBID))
	case strings.TrimSpace(meta.Release.Title) != "":
		params.Set("search", strings.TrimSpace(meta.Release.Title))
	default:
		return nil, []string{noteSkip("missing imdb/title for SPD dupe search")}, nil
	}
	headers := map[string]string{"Authorization": apiKey, "accept": "application/json"}
	status, payload, err := doJSONGetAny(ctx, h.http, "https://speedapp.io/api/torrent", params, headers)
	if err != nil || status < 200 || status >= 300 {
		return nil, []string{noteSkip("SPD search failed")}, nil
	}
	list, ok := anyToSlice(payload)
	if !ok {
		return nil, nil, nil
	}
	entries := make([]api.DupeEntry, 0, len(list))
	for _, raw := range list {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := stringFromAny(item["id"])
		entry := api.DupeEntry{Name: stringFromAny(item["name"]), ID: id, Link: "https://speedapp.io/browse/" + id + "/"}
		size := intFromAny(item["size"])
		if size > 0 {
			entry.SizeKnown = true
			entry.SizeBytes = size
		}
		entries = append(entries, entry)
	}
	return entries, nil, nil
}
