// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteHUNOProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:       resolveUnit3DHUNOTypeID,
		resolveResolutionID: resolveUnit3DHUNOResolutionID,
	}
}

func resolveUnit3DHUNOTypeID(meta api.PreparedMetadata) string {
	typeValue := strings.ToLower(strings.TrimSpace(inferUnit3DType(meta)))
	videoEncode := strings.ToLower(strings.TrimSpace(meta.VideoEncode))

	if typeValue == "remux" {
		return "2"
	}
	if typeValue == "webdl" || typeValue == "webrip" {
		if strings.Contains(videoEncode, "x265") || strings.Contains(videoEncode, "265") {
			return "15"
		}
		return "3"
	}
	if typeValue == "encode" || typeValue == "hdtv" {
		return "15"
	}
	if typeValue == "disc" {
		return "1"
	}
	return "0"
}

func resolveUnit3DHUNOResolutionID(meta api.PreparedMetadata) string {
	resolution := resolveResolution(meta)
	mapping := map[string]string{
		"4320p": "1",
		"2160p": "2",
		"1080p": "3",
		"1080i": "4",
		"720p":  "5",
		"576p":  "6",
		"576i":  "7",
		"540p":  "11",
		"540i":  "11",
		"480p":  "8",
		"480i":  "9",
	}
	if value, ok := mapping[resolution]; ok {
		return value
	}
	return "10"
}
