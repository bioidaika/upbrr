// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import "github.com/autobrr/upbrr/pkg/api"

func siteRFProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:       resolveUnit3DRFTypeID,
		resolveResolutionID: resolveUnit3DRFResolutionID,
	}
}

func resolveUnit3DRFTypeID(meta api.PreparedMetadata) string {
	mapping := map[string]string{
		"DISC":   "43",
		"REMUX":  "40",
		"WEBDL":  "42",
		"WEBRIP": "45",
		"ENCODE": "41",
		"HDTV":   "35",
	}
	return mapping[inferUnit3DType(meta)]
}

func resolveUnit3DRFResolutionID(meta api.PreparedMetadata) string {
	resolution := resolveResolution(meta)
	mapping := map[string]string{
		"4320p": "1",
		"2160p": "2",
		"1080p": "3",
		"1080i": "4",
		"720p":  "5",
		"576p":  "6",
		"576i":  "7",
		"480p":  "8",
		"480i":  "9",
	}
	if value, ok := mapping[resolution]; ok {
		return value
	}
	return "10"
}
