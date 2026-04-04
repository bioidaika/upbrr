// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteIHDProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveResolutionID: resolveUnit3DIHDResolutionID,
		resolveCategoryID:   resolveUnit3DIHDCategoryID,
	}
}

func resolveUnit3DIHDCategoryID(meta api.PreparedMetadata) string {
	category := resolveUnit3DCategory(meta)
	if strings.EqualFold(category, "TV") && meta.Anime {
		return "3"
	}
	if strings.EqualFold(category, "MOVIE") && meta.Anime {
		return "4"
	}
	return resolveUnit3DCategoryID(meta)
}

func resolveUnit3DIHDResolutionID(meta api.PreparedMetadata) string {
	resolution := resolveResolution(meta)
	mapping := map[string]string{
		"4320p": "1",
		"2160p": "2",
		"1440p": "3",
		"1080p": "3",
		"1080i": "4",
	}
	if value, ok := mapping[resolution]; ok {
		return value
	}
	return "10"
}
