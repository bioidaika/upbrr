// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package azfamily

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/autobrr/upbrr/internal/services/db"
)

type simpleCookieJar struct {
	baseURL *url.URL
	cookies map[string]*http.Cookie
}

func (j simpleCookieJar) SetCookies(_ *url.URL, cookies []*http.Cookie) {
	if j.cookies == nil {
		return
	}
	for _, cookie := range cookies {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		j.cookies[cookie.Name] = cookie
	}
}

func (j simpleCookieJar) Cookies(_ *url.URL) []*http.Cookie {
	if j.cookies == nil {
		return nil
	}
	out := make([]*http.Cookie, 0, len(j.cookies))
	for _, cookie := range j.cookies {
		out = append(out, cookie)
	}
	return out
}

func resolveCookies(dbPath string, site siteDefinition) ([]*http.Cookie, error) {
	path := resolveCookiePath(dbPath, site.Name)
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("trackers: %s cookie file not found", site.Name)
	}
	return loadNetscapeCookies(path, mustParseURL(site.BaseURL).Hostname())
}

func resolveCookiePath(dbPath string, tracker string) string {
	candidates := make([]string, 0, 1)
	if strings.TrimSpace(dbPath) != "" {
		if path, err := db.CookiePath(dbPath, tracker+".txt"); err == nil {
			candidates = append(candidates, path)
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func loadNetscapeCookies(path string, expectedDomain string) ([]*http.Cookie, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	target := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(expectedDomain)), ".")
	out := make([]*http.Cookie, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "#\t") {
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
		domain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(fields[0])), ".")
		if target != "" && !strings.Contains(domain, target) {
			continue
		}
		out = append(out, &http.Cookie{
			Domain: strings.TrimSpace(fields[0]),
			Path:   strings.TrimSpace(fields[2]),
			Secure: strings.EqualFold(strings.TrimSpace(fields[3]), "TRUE"),
			Name:   strings.TrimSpace(fields[5]),
			Value:  strings.TrimSpace(fields[6]),
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no valid cookies found")
	}
	return out, nil
}
