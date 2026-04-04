// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package imdb

import "testing"

func TestRankCandidates(t *testing.T) {
	results := []map[string]any{
		{
			"node": map[string]any{
				"title": map[string]any{
					"id":          "tt0000001",
					"titleText":   map[string]any{"text": "Example Title"},
					"releaseYear": map[string]any{"year": 2020},
					"titleType":   map[string]any{"text": "Movie"},
					"plot":        map[string]any{"plotText": map[string]any{"plainText": "Plot"}},
				},
			},
		},
		{
			"node": map[string]any{
				"title": map[string]any{
					"id":          "tt0000002",
					"titleText":   map[string]any{"text": "Other Title"},
					"releaseYear": map[string]any{"year": 2020},
					"titleType":   map[string]any{"text": "Movie"},
					"plot":        map[string]any{"plotText": map[string]any{"plainText": "Plot 2"}},
				},
			},
		},
	}
	candidates := rankCandidates(results, "Example Title", 2020)
	if len(candidates) == 0 {
		t.Fatalf("expected candidates")
	}
	if candidates[0].IMDbID != 1 {
		t.Fatalf("expected IMDbID 1, got %d", candidates[0].IMDbID)
	}
}
