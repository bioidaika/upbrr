// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/autobrr/upbrr/internal/config"
	ascimpl "github.com/autobrr/upbrr/internal/trackers/impl/asc"
	"github.com/autobrr/upbrr/pkg/api"
)

type ascHandler struct {
	cfg    config.Config
	http   *http.Client
	logger api.Logger
}

func (h ascHandler) Search(ctx context.Context, meta api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	if h.http == nil {
		return nil, []string{noteSkip("ASC handler misconfigured: no HTTP client")}, nil
	}
	if !meta.Anime && resolveASCIMDb(meta) == "" {
		return nil, []string{noteSkip("missing IMDb ID for ASC dupe search")}, nil
	}

	cookies, _, err := ascimpl.LoadCookies(h.cfg.MainSettings.DBPath)
	if err != nil || len(cookies) == 0 {
		return nil, []string{noteSkip("missing valid ASC cookies (expected Netscape cookies at cookies/ASC.txt)")}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildASCSearchURL(meta), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build ASC request: %w", err)
	}
	req.Header.Set("User-Agent", "upbrr")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := h.http.Do(req)
	if err != nil {
		return nil, []string{noteSkip("ASC request failed")}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, []string{noteSkip("ASC search returned non-success status")}, nil
	}

	entries, err := parseASCSearchResults(resp.Body)
	if err != nil {
		return nil, []string{noteSkip("ASC response parse failed")}, nil
	}
	return entries, nil, nil
}

func buildASCSearchURL(meta api.PreparedMetadata) string {
	base := "https://cliente.amigos-share.club"
	if meta.Anime {
		return base + "/torrents-search.php?search=" + url.QueryEscape(resolveASCTitle(meta))
	}
	if strings.EqualFold(resolveASCCategory(meta), "TV") {
		return base + "/busca-series.php?search=" + url.QueryEscape(resolveASCSeasonEpisode(meta)) + "&imdb=" + url.QueryEscape(resolveASCIMDb(meta))
	}
	return base + "/busca-filmes.php?search=&imdb=" + url.QueryEscape(resolveASCIMDb(meta))
}

func parseASCSearchResults(body io.Reader) ([]api.DupeEntry, error) {
	tokenizer := html.NewTokenizer(body)
	entries := make([]api.DupeEntry, 0)
	seen := map[string]struct{}{}

	for {
		//nolint:exhaustive // We only care about tokens relevant to ASC result links.
		switch tokenizer.Next() {
		case html.ErrorToken:
			return entries, nil
		case html.StartTagToken:
			token := tokenizer.Token()
			if !strings.EqualFold(token.Data, "a") {
				continue
			}
			href := ""
			for _, attr := range token.Attr {
				if strings.EqualFold(attr.Key, "href") {
					href = strings.TrimSpace(attr.Val)
					break
				}
			}
			if !strings.Contains(href, "torrents-details.php?id=") {
				continue
			}
			name := strings.TrimSpace(readASCAnchorText(tokenizer))
			id := parseASCTorrentID(href)
			key := firstNonEmpty(id, href, name)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			entries = append(entries, api.DupeEntry{
				Name: name,
				ID:   id,
				Link: absolutizeASCLink(href),
			})
		}
	}
}

func readASCAnchorText(tokenizer *html.Tokenizer) string {
	var builder strings.Builder
	depth := 1
	for depth > 0 {
		//nolint:exhaustive // Only text and anchor boundaries matter for ASC titles.
		switch tokenizer.Next() {
		case html.ErrorToken:
			return builder.String()
		case html.StartTagToken:
			depth++
		case html.EndTagToken:
			depth--
		case html.TextToken:
			_, _ = builder.Write(tokenizer.Text())
		}
	}
	return builder.String()
}

func parseASCTorrentID(href string) string {
	parsed, err := url.Parse(href)
	if err == nil {
		if id := strings.TrimSpace(parsed.Query().Get("id")); id != "" {
			return id
		}
	}
	parts := strings.Split(href, "id=")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(strings.Split(parts[len(parts)-1], "&")[0])
}

func absolutizeASCLink(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	return "https://cliente.amigos-share.club/" + strings.TrimPrefix(href, "/")
}

func resolveASCIMDb(meta api.PreparedMetadata) string {
	if meta.ExternalMetadata.IMDB != nil && strings.TrimSpace(meta.ExternalMetadata.IMDB.IMDbIDText) != "" {
		return strings.TrimSpace(meta.ExternalMetadata.IMDB.IMDbIDText)
	}
	if meta.ExternalIDs.IMDBID > 0 {
		return fmt.Sprintf("tt%07d", meta.ExternalIDs.IMDBID)
	}
	return ""
}

func resolveASCCategory(meta api.PreparedMetadata) string {
	if strings.EqualFold(meta.ExternalIDs.Category, "TV") || meta.TVPack || meta.SeasonInt > 0 || meta.EpisodeInt > 0 {
		return "TV"
	}
	return "MOVIE"
}

func resolveASCTitle(meta api.PreparedMetadata) string {
	if strings.TrimSpace(meta.Release.Title) != "" {
		return strings.TrimSpace(meta.Release.Title)
	}
	if strings.TrimSpace(meta.ReleaseName) != "" {
		return strings.TrimSpace(meta.ReleaseName)
	}
	return strings.TrimSpace(meta.SourcePath)
}

func resolveASCSeasonEpisode(meta api.PreparedMetadata) string {
	if meta.EpisodeInt > 0 {
		return "S" + padASC2(meta.SeasonInt) + "E" + padASC2(meta.EpisodeInt)
	}
	if meta.SeasonInt > 0 {
		return "S" + padASC2(meta.SeasonInt)
	}
	return ""
}

func padASC2(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
