// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/pkg/api"
)

type nblHandler struct {
	cfg  config.Config
	http *http.Client
}

func (h nblHandler) Search(ctx context.Context, meta api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	_, apiKey, ok := trackerCfgWithAPIKey(h.cfg, "NBL")
	if !ok {
		return nil, []string{noteSkip("missing api_key for tracker")}, nil
	}
	searchTerm := map[string]any{}
	switch {
	case meta.ExternalIDs.TVmazeID != 0:
		searchTerm["tvmaze"] = meta.ExternalIDs.TVmazeID
	case meta.ExternalIDs.IMDBID != 0:
		searchTerm["imdb"] = strconv.Itoa(meta.ExternalIDs.IMDBID)
	default:
		searchTerm["series"] = strings.TrimSpace(meta.Release.Title)
	}
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTorrents",
		"params":  []any{apiKey, searchTerm},
	}
	status, body, err := doJSONPost(ctx, h.http, "https://nebulance.io/api.php", payload, nil)
	if err != nil || status < 200 || status >= 300 || len(body) == 0 {
		return nil, []string{noteSkip("NBL search failed")}, nil
	}
	result, _ := body["result"].(map[string]any)
	items, _ := result["items"].([]any)
	entries := make([]api.DupeEntry, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		entry := api.DupeEntry{
			Name: stringFromAny(item["rls_name"]),
			Link: "https://nebulance.io/torrents.php?id=" + stringFromAny(item["group_id"]),
		}
		size := intFromAny(item["size"])
		if size > 0 {
			entry.SizeKnown = true
			entry.SizeBytes = size
		}
		if fileList, ok := item["file_list"].([]any); ok {
			entry.FileCount = len(fileList)
			for _, file := range fileList {
				f := stringFromAny(file)
				if f != "" {
					entry.Files = append(entry.Files, f)
				}
			}
		}
		entry.Download = stringFromAny(item["download"])
		entries = append(entries, entry)
	}
	return entries, nil, nil
}
