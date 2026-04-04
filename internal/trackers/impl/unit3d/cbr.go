// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteCBRProfile() unit3DSiteProfile {
	return unit3DSiteProfile{resolveCategoryID: resolveUnit3DCBRCategoryID}
}

func resolveUnit3DCBRCategoryID(meta api.PreparedMetadata) string {
	if strings.EqualFold(resolveUnit3DCategory(meta), "TV") && meta.Anime {
		return "4"
	}
	return resolveUnit3DCategoryID(meta)
}
