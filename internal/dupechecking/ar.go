// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/services/db"
	"github.com/autobrr/upbrr/pkg/api"
)

const (
	arBrowseEndpoint = "https://alpharatio.cc/ajax.php"
	arCookieFileName = "AR.txt"
)

type arHandler struct {
	cfg    config.Config
	http   *http.Client
	logger api.Logger
}

func (h arHandler) Search(ctx context.Context, meta api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	if h.http == nil {
		return nil, []string{noteSkip("AR handler misconfigured: no HTTP client")}, nil
	}

	query := arSearchQuery(meta)
	if query == "" {
		return nil, []string{noteSkip("missing title for AR dupe search")}, nil
	}

	cookies, cookiePath, err := h.resolveCookies()
	if err != nil || len(cookies) == 0 {
		if err != nil && h.logger != nil {
			h.logger.Debugf("dupechecking: AR cookie resolution failed: %v", err)
		}
		return nil, []string{noteSkip("missing valid AR cookies (expected Netscape cookies at cookies/AR.txt)")}, nil
	}
	if h.logger != nil && cookiePath != "" {
		h.logger.Debugf("dupechecking: AR using cookie file %s", cookiePath)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, arBrowseEndpoint, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build AR request: %w", err)
	}
	params := req.URL.Query()
	params.Set("action", "browse")
	params.Set("searchstr", query)
	req.URL.RawQuery = params.Encode()
	req.Header.Set("User-Agent", "upbrr")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := h.http.Do(req)
	if err != nil {
		if h.logger != nil {
			h.logger.Debugf("dupechecking: AR request failed: %v", err)
		}
		return nil, []string{noteSkip("AR request failed")}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if h.logger != nil {
			h.logger.Debugf("dupechecking: AR returned status %d", resp.StatusCode)
		}
		return nil, []string{noteSkip("AR search returned non-success status")}, nil
	}

	var payload arResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		if h.logger != nil {
			h.logger.Debugf("dupechecking: AR response decode failed: %v", err)
		}
		return nil, []string{noteSkip("AR response decode failed")}, nil
	}
	if !strings.EqualFold(strings.TrimSpace(payload.Status), "success") {
		return nil, []string{noteSkip("AR API returned non-success status")}, nil
	}

	entries := make([]api.DupeEntry, 0, len(payload.Response.Results))
	for _, result := range payload.Response.Results {
		name := strings.TrimSpace(result.GroupName)
		if name == "" {
			continue
		}
		entry := api.DupeEntry{
			Name:      name,
			Files:     []string{name},
			FileCount: result.FileCount,
			ID:        strconv.FormatInt(result.TorrentID, 10),
			Link:      "https://alpharatio.cc/torrents.php?id=" + strconv.FormatInt(result.GroupID, 10) + "&torrentid=" + strconv.FormatInt(result.TorrentID, 10),
			Download:  "https://alpharatio.cc/torrents.php?action=download&id=" + strconv.FormatInt(result.TorrentID, 10),
		}
		if result.Size > 0 {
			entry.SizeKnown = true
			entry.SizeBytes = result.Size
		}
		entries = append(entries, entry)
	}

	return entries, nil, nil
}

func (h arHandler) resolveCookies() ([]*http.Cookie, string, error) {
	arURL, _ := url.Parse("https://alpharatio.cc/")
	merged := map[string]*http.Cookie{}

	if h.http != nil && h.http.Jar != nil && arURL != nil {
		for _, cookie := range h.http.Jar.Cookies(arURL) {
			if cookie == nil || strings.TrimSpace(cookie.Name) == "" {
				continue
			}
			merged[cookie.Name] = cookie
		}
	}
	if len(merged) > 0 {
		return mapCookiesToSlice(merged), "", nil
	}

	cookiePath, err := resolveARCookiePath(h.cfg)
	if err != nil {
		return nil, "", err
	}
	loaded, err := loadNetscapeCookies(cookiePath, "alpharatio.cc")
	if err != nil {
		return nil, "", err
	}
	for _, cookie := range loaded {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		merged[cookie.Name] = cookie
	}
	if len(merged) == 0 {
		return nil, "", errors.New("no valid cookies found")
	}
	return mapCookiesToSlice(merged), cookiePath, nil
}

func mapCookiesToSlice(values map[string]*http.Cookie) []*http.Cookie {
	if len(values) == 0 {
		return nil
	}
	out := make([]*http.Cookie, 0, len(values))
	for _, cookie := range values {
		out = append(out, cookie)
	}
	return out
}

func resolveARCookiePath(cfg config.Config) (string, error) {
	candidates := arCookiePathCandidates(cfg)
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return candidate, nil
	}
	return "", errors.New("AR cookie file not found")
}

func arCookiePathCandidates(cfg config.Config) []string {
	candidates := make([]string, 0, 1)
	seen := map[string]struct{}{}
	add := func(path string) {
		cleaned := strings.TrimSpace(filepath.Clean(path))
		if cleaned == "" {
			return
		}
		if _, exists := seen[cleaned]; exists {
			return
		}
		seen[cleaned] = struct{}{}
		candidates = append(candidates, cleaned)
	}

	if dbPath := strings.TrimSpace(cfg.MainSettings.DBPath); dbPath != "" {
		if path, err := db.CookiePath(dbPath, arCookieFileName); err == nil {
			add(path)
		}
	}
	return candidates
}

func loadNetscapeCookies(path string, expectedDomain string) ([]*http.Cookie, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	cookies := make([]*http.Cookie, 0)
	targetDomain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(expectedDomain)), ".")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#HttpOnly_") {
			line = strings.TrimPrefix(line, "#HttpOnly_")
		} else if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}

		domain := strings.TrimPrefix(strings.TrimSpace(fields[0]), ".")
		domainLower := strings.ToLower(domain)
		if domain == "" {
			continue
		}
		if targetDomain != "" && domainLower != targetDomain && !strings.HasSuffix(domainLower, "."+targetDomain) {
			continue
		}
		name := strings.TrimSpace(fields[5])
		value := strings.TrimSpace(strings.Join(fields[6:], "\t"))
		if name == "" || value == "" {
			continue
		}
		pathValue := strings.TrimSpace(fields[2])
		if pathValue == "" {
			pathValue = "/"
		}
		secure := strings.EqualFold(strings.TrimSpace(fields[3]), "TRUE")
		cookies = append(cookies, &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: domain,
			Path:   pathValue,
			Secure: secure,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cookies, nil
}

func arSearchQuery(meta api.PreparedMetadata) string {
	title := strings.TrimSpace(meta.Release.Title)
	if title == "" && meta.ExternalMetadata.TMDB != nil {
		title = strings.TrimSpace(meta.ExternalMetadata.TMDB.Title)
	}
	if title == "" && meta.ExternalMetadata.IMDB != nil {
		title = strings.TrimSpace(meta.ExternalMetadata.IMDB.Title)
	}
	if title == "" {
		title = strings.TrimSpace(meta.ReleaseName)
	}
	if title == "" {
		return ""
	}

	year := meta.Release.Year
	if year == 0 && meta.ExternalMetadata.TMDB != nil && meta.ExternalMetadata.TMDB.Year > 0 {
		year = meta.ExternalMetadata.TMDB.Year
	}
	if year == 0 && meta.ExternalMetadata.IMDB != nil && meta.ExternalMetadata.IMDB.Year > 0 {
		year = meta.ExternalMetadata.IMDB.Year
	}
	if year > 0 {
		return strings.TrimSpace(title + " " + strconv.Itoa(year))
	}
	return title
}

type arResponse struct {
	Status   string `json:"status"`
	Response struct {
		Results []struct {
			GroupName string `json:"groupName"`
			Size      int64  `json:"size"`
			FileCount int    `json:"fileCount"`
			GroupID   int64  `json:"groupId"`
			TorrentID int64  `json:"torrentId"`
		} `json:"results"`
	} `json:"response"`
}
