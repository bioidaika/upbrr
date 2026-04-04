// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteTOSProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:     resolveUnit3DTOSTypeID,
		resolveCategoryID: resolveUnit3DTOSCategoryID,
	}
}

func resolveUnit3DTOSCategoryID(meta api.PreparedMetadata) string {
	tag := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(meta.Tag, "-")))
	isSubFrench := strings.Contains(tag, "vostfr") || strings.Contains(tag, "subfrench")
	category := resolveUnit3DCategory(meta)
	if strings.EqualFold(category, "TV") {
		if meta.TVPack {
			if isSubFrench {
				return "9"
			}
			return "8"
		}
		if isSubFrench {
			return "7"
		}
		return "2"
	}
	if isSubFrench {
		return "6"
	}
	return "1"
}

func resolveUnit3DTOSTypeID(meta api.PreparedMetadata) string {
	if strings.EqualFold(strings.TrimSpace(meta.DiscType), "DVD") {
		return "7"
	}
	if strings.EqualFold(strings.TrimSpace(meta.Is3D), "3D") {
		return "8"
	}
	mapping := map[string]string{
		"DISC":   "1",
		"REMUX":  "2",
		"ENCODE": "3",
		"WEBDL":  "4",
		"WEBRIP": "5",
		"HDTV":   "6",
	}
	return mapping[inferUnit3DType(meta)]
}
