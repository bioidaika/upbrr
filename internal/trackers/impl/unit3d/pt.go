// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/pkg/api"
)

func sitePTProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:       resolveUnit3DPTTypeID,
		resolveResolutionID: resolveUnit3DPTResolutionID,
		applyAdditionalPayload: func(req trackers.UploadRequest, data map[string]string) {
			data["audio_pt"] = boolFlag(hasEuropeanPortuguese(req.Meta.AudioLanguages))
			data["legenda_pt"] = boolFlag(hasEuropeanPortuguese(req.Meta.SubtitleLanguages))
		},
	}
}

func hasEuropeanPortuguese(languages []string) bool {
	for _, language := range languages {
		lower := strings.ToLower(strings.TrimSpace(language))
		if lower == "" {
			continue
		}
		if strings.Contains(lower, "brazil") || strings.Contains(lower, "brasil") || strings.Contains(lower, "pt-br") || strings.Contains(lower, "ptbr") {
			continue
		}
		if strings.Contains(lower, "portuguese") || strings.EqualFold(lower, "pt") || strings.Contains(lower, "português") {
			return true
		}
	}
	return false
}

func resolveUnit3DPTTypeID(meta api.PreparedMetadata) string {
	mapping := map[string]string{
		"DISC":   "1",
		"REMUX":  "2",
		"WEBDL":  "4",
		"WEBRIP": "39",
		"HDTV":   "6",
		"ENCODE": "3",
	}
	return mapping[inferUnit3DType(meta)]
}

func resolveUnit3DPTResolutionID(meta api.PreparedMetadata) string {
	resolution := resolveResolution(meta)
	mapping := map[string]string{
		"4320p": "1",
		"2160p": "2",
		"1440p": "13",
		"1080p": "3",
		"1080i": "4",
		"720p":  "5",
		"576p":  "6",
		"576i":  "7",
		"540p":  "11",
		"480p":  "8",
		"480i":  "9",
	}
	if value, ok := mapping[resolution]; ok {
		return value
	}
	return "10"
}
