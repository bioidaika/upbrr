// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteOTWProfile() unit3DSiteProfile {
	return unit3DSiteProfile{resolveTypeID: resolveUnit3DOTWTypeID}
}

func resolveUnit3DOTWTypeID(meta api.PreparedMetadata) string {
	if strings.EqualFold(strings.TrimSpace(meta.DiscType), "BDMV") {
		return "1"
	}
	if isDiscType(meta.DiscType) {
		return "7"
	}

	typeValue := inferUnit3DType(meta)
	if typeValue == "DVDRIP" {
		return "8"
	}

	mapping := map[string]string{
		"DISC":   "1",
		"REMUX":  "2",
		"WEBDL":  "4",
		"WEBRIP": "5",
		"HDTV":   "6",
		"ENCODE": "3",
	}
	return mapping[typeValue]
}
