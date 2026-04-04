// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metadata

import (
	"strings"

	"github.com/autobrr/upbrr/internal/pathutil"
	"github.com/autobrr/upbrr/pkg/api"
)

func resolveService(meta api.PreparedMetadata) (string, string, string) {
	services := serviceCodeMap()
	filename := strings.TrimSpace(meta.Filename)
	if filename == "" {
		filename = pathutil.Base(meta.SourcePath)
	}
	cleaned := strings.NewReplacer(".", " ", "(", " ", ")", " ").Replace(filename)
	if tag := strings.TrimSpace(meta.Tag); tag != "" {
		cleaned = strings.ReplaceAll(cleaned, tag, "")
	}
	if audio := strings.TrimSpace(meta.Audio); strings.Contains(audio, "DTS-HD MA") {
		cleaned = strings.ReplaceAll(cleaned, "DTS-HD.MA.", "")
		cleaned = strings.ReplaceAll(cleaned, "DTS-HD MA ", "")
	}
	cleanedLower := " " + strings.ToLower(cleaned) + " "

	service := strings.TrimSpace(meta.Service)
	if service == "" {
		for key, value := range services {
			needle := " " + strings.ToLower(strings.TrimSpace(key)) + " "
			if needle != "  " && strings.Contains(cleanedLower, needle) {
				service = value
			}
		}
	}

	longName := ""
	if service != "" {
		for key, value := range services {
			if value == service {
				if len(key) > len(longName) {
					longName = key
				}
			}
		}
		if longName == "" {
			longName = service
		}
	}

	return service, longName, filename
}
