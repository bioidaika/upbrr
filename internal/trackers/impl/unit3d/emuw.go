// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import "github.com/autobrr/upbrr/pkg/api"

func siteEMUWProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:       resolveUnit3DEMUWTypeID,
		resolveResolutionID: resolveUnit3DEMUWResolutionID,
	}
}

func resolveUnit3DEMUWTypeID(meta api.PreparedMetadata) string {
	mapping := map[string]string{
		"DISC":   "1",
		"REMUX":  "2",
		"ENCODE": "3",
		"WEBDL":  "4",
		"WEBRIP": "5",
		"HDTV":   "6",
	}
	typeID := mapping[inferUnit3DType(meta)]
	if typeID == "" {
		return "3"
	}
	return typeID
}

func resolveUnit3DEMUWResolutionID(meta api.PreparedMetadata) string {
	resolution := resolveResolution(meta)
	mapping := map[string]string{
		"4320p": "1",
		"2160p": "2",
		"1080p": "3",
		"1080i": "4",
		"720p":  "5",
		"576p":  "6",
		"540p":  "7",
		"480p":  "8",
	}
	if value, ok := mapping[resolution]; ok {
		return value
	}
	return "10"
}
