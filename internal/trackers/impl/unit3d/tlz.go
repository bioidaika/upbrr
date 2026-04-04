// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteTLZProfile() unit3DSiteProfile {
	return unit3DSiteProfile{resolveTypeID: resolveUnit3DTLZTypeID}
}

func resolveUnit3DTLZTypeID(meta api.PreparedMetadata) string {
	if strings.EqualFold(resolveUnit3DCategory(meta), "MOVIE") {
		return "1"
	}
	if meta.TVPack {
		return "4"
	}
	return "3"
}
