// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	xhtml "golang.org/x/net/html"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/metadata/metautil"
	"github.com/autobrr/upbrr/pkg/api"
)

var (
	nethdSEONameFirstPattern = regexp.MustCompile(`(?i)(?:^|/)([^/?#]+)-torrent-(\d+)\.html$`)
	nethdSEOIDFirstPattern   = regexp.MustCompile(`(?i)(?:^|/)torrent-(\d+)(?:-([^/?#]+))?\.html$`)
	nethdSizePattern         = regexp.MustCompile(`(?i)\d+(?:\.\d+)?\s*[kmgt]i?b`)
)

type nethdHandler struct {
	cfg  config.Config
	http *http.Client
}

func (h nethdHandler) Search(ctx context.Context, meta api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	if h.http == nil {
		return nil, []string{noteSkip("NETHD handler misconfigured: no HTTP client")}, nil
	}

	searchTerm := imdbForLookup(meta)
	searchArea := "4"
	if searchTerm == "" {
		searchTerm = metautil.FirstNonEmptyTrimmed(meta.Release.Title, meta.ReleaseName)
		searchArea = "0"
	}
	if searchTerm == "" {
		return nil, []string{noteSkip("missing IMDb ID or title for NETHD dupe search")}, nil
	}

	baseURL := trackerBaseURL(h.cfg, "NETHD", "https://nethd.org")
	values, err := loadTrackerCookies(ctx, h.cfg, "NETHD", trackerHost(baseURL, "nethd.org"))
	if err != nil {
		return nil, []string{noteSkip("missing valid NETHD cookies")}, nil
	}
	params := url.Values{
		"incldead":    {"0"},
		"search":      {searchTerm},
		"search_area": {searchArea},
		"search_mode": {"0"},
	}
	resp, root, err := doHTMLGet(ctx, h.http, baseURL+"/torrents.php", params, values)
	if err != nil || !resp.ok() {
		return nil, []string{noteSkip("NETHD search failed")}, nil
	}

	rows := findNodes(root, func(node *xhtml.Node) bool {
		return node.Type == xhtml.ElementNode && node.Data == "tr"
	})
	entries := make([]api.DupeEntry, 0, len(rows))
	for _, row := range rows {
		linkNode, id, slug := nethdDetailLink(row)
		if linkNode == nil {
			continue
		}

		name := strings.TrimSpace(nodeTextHTML(linkNode))
		if title := strings.TrimSpace(attrValueHTML(linkNode, "title")); len(title) > len(name) {
			name = title
		}
		slugName := strings.TrimSpace(strings.ReplaceAll(slug, "-", " "))
		if len(slugName) > len(name) || (detectResolution(slugName) != "" && detectResolution(name) == "") {
			name = slugName
		}
		if name == "" {
			continue
		}

		entry := api.DupeEntry{
			ID:   id,
			Name: name,
			Link: absoluteURL(baseURL, attrValueHTML(linkNode, "href")),
		}
		cells := findNodes(row, func(node *xhtml.Node) bool {
			return node.Type == xhtml.ElementNode && node.Data == "td"
		})
		for _, cell := range cells {
			if size := nethdSizePattern.FindString(strings.TrimSpace(nodeTextHTML(cell))); size != "" {
				addSize(&entry, size)
				break
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil, nil
}

// nethdDetailLink accepts NexusPHP's query-string detail link plus both SEO
// layouts observed in NETHD responses: name-torrent-ID and torrent-ID-name.
func nethdDetailLink(root *xhtml.Node) (*xhtml.Node, string, string) {
	links := findNodes(root, func(node *xhtml.Node) bool {
		return node.Type == xhtml.ElementNode && node.Data == "a"
	})
	for _, link := range links {
		id, slug, ok := parseNETHDDetailHref(attrValueHTML(link, "href"))
		if ok {
			return link, id, slug
		}
	}
	return nil, "", ""
}

func parseNETHDDetailHref(href string) (id string, slug string, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return "", "", false
	}
	path := strings.TrimSpace(parsed.Path)
	lowerPath := strings.ToLower(strings.TrimPrefix(path, "/"))
	if lowerPath == "details.php" || strings.HasSuffix(lowerPath, "/details.php") {
		if id := strings.TrimSpace(parsed.Query().Get("id")); isDecimalID(id) {
			return id, "", true
		}
	}
	if match := nethdSEONameFirstPattern.FindStringSubmatch(path); len(match) == 3 {
		return match[2], match[1], true
	}
	if match := nethdSEOIDFirstPattern.FindStringSubmatch(path); len(match) == 3 {
		return match[1], match[2], true
	}
	return "", "", false
}

func isDecimalID(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
