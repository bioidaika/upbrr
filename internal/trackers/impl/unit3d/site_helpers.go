// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func resolveTMDBGenres(meta api.PreparedMetadata) string {
	if meta.ExternalMetadata.TMDB != nil {
		return strings.TrimSpace(meta.ExternalMetadata.TMDB.Genres)
	}
	return ""
}

func resolveIMDBGenres(meta api.PreparedMetadata) string {
	if meta.ExternalMetadata.IMDB != nil {
		return strings.TrimSpace(meta.ExternalMetadata.IMDB.Genres)
	}
	return ""
}

func hasAdultToken(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	for _, token := range []string{"adult", "porn", "pornography", "xxx", "erotic", "hentai"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func hasEnglishLanguage(languages []string) bool {
	for _, language := range languages {
		lower := strings.ToLower(strings.TrimSpace(language))
		switch lower {
		case "english", "en", "eng", "en-us", "en-gb":
			return true
		}
	}
	return false
}
