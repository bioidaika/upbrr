// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func BuildAitherName(meta api.PreparedMetadata) string {
	name := strings.TrimSpace(meta.ReleaseName)
	if name == "" {
		name = strings.TrimSpace(meta.ReleaseNameNoTag)
	}
	if name == "" {
		return ""
	}

	resolution := resolveResolution(meta)
	videoCodec := strings.TrimSpace(meta.VideoCodec)
	videoEncode := strings.TrimSpace(meta.VideoEncode)
	nameType := strings.ToUpper(strings.TrimSpace(meta.Type))
	source := strings.TrimSpace(meta.Source)
	audio := strings.TrimSpace(meta.Audio)

	languages := append([]string{}, meta.Release.Language...)
	if len(languages) > 0 && !hasEnglishLanguage(languages) {
		foreignLang := strings.ToUpper(strings.TrimSpace(languages[0]))
		if nameType == "REMUX" && isDVDSource(source) {
			if meta.Release.Year > 0 {
				name = strings.Replace(name, formatOptionalInt(meta.Release.Year), formatOptionalInt(meta.Release.Year)+" "+foreignLang, 1)
			}
		} else if !strings.EqualFold(strings.TrimSpace(meta.DiscType), "BDMV") {
			if resolution != "" {
				name = strings.Replace(name, resolution, foreignLang+" "+resolution, 1)
			}
		}
	}

	if nameType == "DVDRIP" {
		source = "DVDRip"
		if meta.Source != "" {
			name = strings.Replace(name, meta.Source+" ", "", 1)
		}
		if videoEncode != "" {
			name = strings.Replace(name, videoEncode, "", 1)
		}
		if resolution != "" {
			name = strings.Replace(name, source, resolution+" "+source, 1)
		}
		if audio != "" && videoEncode != "" {
			name = strings.Replace(name, audio, audio+videoEncode, 1)
		}
	} else if strings.EqualFold(strings.TrimSpace(meta.DiscType), "DVD") || (nameType == "REMUX" && isDVDSource(source)) {
		if resolution != "" && source != "" {
			name = strings.Replace(name, source, resolution+" "+source, 1)
		}
		if audio != "" && videoCodec != "" {
			name = strings.Replace(name, audio, videoCodec+" "+audio, 1)
		}
	}

	return strings.TrimSpace(strings.Join(strings.Fields(name), " "))
}

func isDVDSource(source string) bool {
	value := strings.ToUpper(strings.TrimSpace(source))
	switch value {
	case "PAL DVD", "NTSC DVD", "DVD":
		return true
	default:
		return false
	}
}
