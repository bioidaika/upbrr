// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package asc

import (
	"bufio"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobrr/upbrr/internal/services/db"
)

const (
	cookieFileName = "ASC.txt"
	baseURL        = "https://cliente.amigos-share.club"
	userAgent      = "upbrr"
	sourceFlag     = "ASC"
)

func LoadCookies(dbPath string) ([]*http.Cookie, string, error) {
	for _, candidate := range cookiePathCandidates(dbPath) {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		cookies, err := loadNetscapeCookies(candidate, "cliente.amigos-share.club")
		if err != nil {
			return nil, "", err
		}
		return cookies, candidate, nil
	}
	return nil, "", errors.New("ASC cookie file not found")
}

func cookiePathCandidates(dbPath string) []string {
	candidates := make([]string, 0, 1)
	seen := map[string]struct{}{}
	add := func(path string) {
		cleaned := strings.TrimSpace(filepath.Clean(path))
		if cleaned == "" {
			return
		}
		if _, ok := seen[cleaned]; ok {
			return
		}
		seen[cleaned] = struct{}{}
		candidates = append(candidates, cleaned)
	}
	if strings.TrimSpace(dbPath) != "" {
		if path, err := db.CookiePath(dbPath, cookieFileName); err == nil {
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

	target := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(expectedDomain)), ".")
	scanner := bufio.NewScanner(file)
	cookies := make([]*http.Cookie, 0, 4)
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
		domain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(fields[0])), ".")
		if domain == "" {
			continue
		}
		if target != "" && domain != target && !strings.HasSuffix(domain, "."+target) {
			continue
		}
		name := strings.TrimSpace(fields[5])
		value := strings.TrimSpace(strings.Join(fields[6:], "\t"))
		if name == "" || value == "" {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Domain: "." + domain,
			Path:   firstNonEmpty(strings.TrimSpace(fields[2]), "/"),
			Secure: strings.EqualFold(strings.TrimSpace(fields[3]), "TRUE"),
			Name:   name,
			Value:  value,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(cookies) == 0 {
		return nil, errors.New("no valid cookies found")
	}
	return cookies, nil
}

func authProblem(dbPath string) string {
	cookies, _, err := LoadCookies(dbPath)
	if err == nil && len(cookies) > 0 {
		return ""
	}
	return "missing valid ASC cookies (expected Netscape cookies at cookies/ASC.txt)"
}
