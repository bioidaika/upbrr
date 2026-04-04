// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteITTProfile() unit3DSiteProfile {
	return unit3DSiteProfile{resolveTypeID: resolveUnit3DITTTypeID}
}

func resolveUnit3DITTTypeID(meta api.PreparedMetadata) string {
	name := strings.ToUpper(strings.TrimSpace(meta.ReleaseName))
	switch {
	case strings.Contains(name, "DLMUX"):
		return "27"
	case strings.Contains(name, "BDMUX"):
		return "29"
	case strings.Contains(name, "WEBMUX"):
		return "26"
	case strings.Contains(name, "DVDMUX"):
		return "39"
	case strings.Contains(name, "BDRIP"):
		return "25"
	}

	mapping := map[string]string{
		"DISC":      "1",
		"REMUX":     "2",
		"WEBDL":     "4",
		"WEBRIP":    "5",
		"HDTV":      "6",
		"ENCODE":    "3",
		"DVDRIP":    "24",
		"CINEMA-MD": "14",
	}
	return mapping[inferUnit3DType(meta)]
}
