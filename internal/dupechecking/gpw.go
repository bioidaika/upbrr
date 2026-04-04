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

type gpwHandler struct {
	cfg  config.Config
	http *http.Client
}

func (h gpwHandler) Search(ctx context.Context, meta api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	_, apiKey, ok := trackerCfgWithAPIKey(h.cfg, "GPW")
	if !ok {
		return nil, []string{noteSkip("missing api_key for tracker")}, nil
	}
	if meta.ExternalIDs.IMDBID == 0 {
		return nil, []string{noteSkip("missing imdb id for GPW dupe search")}, nil
	}
	endpoint := "https://greatposterwall.com/api.php"
	params := url.Values{}
	params.Set("api_key", apiKey)
	params.Set("action", "torrent")
	params.Set("imdbID", "tt"+strconv.Itoa(meta.ExternalIDs.IMDBID))
	status, payload, err := doJSONGet(ctx, h.http, endpoint, params, nil)
	if err != nil || status < 200 || status >= 300 {
		return nil, []string{noteSkip("GPW search failed")}, nil
	}
	if intFromAny(payload["status"]) != 200 {
		return nil, nil, nil
	}
	resp, _ := payload["response"].([]any)
	entries := make([]api.DupeEntry, 0, len(resp))
	for _, raw := range resp {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(strings.Join([]string{
			stringFromAny(item["Name"]),
			stringFromAny(item["Year"]),
			stringFromAny(item["Resolution"]),
			stringFromAny(item["Source"]),
			stringFromAny(item["Processing"]),
			stringFromAny(item["RemasterTitle"]),
			stringFromAny(item["Codec"]),
		}, " "))
		entries = append(entries, api.DupeEntry{Name: strings.Join(strings.Fields(name), " ")})
	}
	return entries, nil, nil
}
