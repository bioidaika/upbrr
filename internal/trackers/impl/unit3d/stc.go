// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import "github.com/autobrr/upbrr/pkg/api"

func siteSTCProfile() unit3DSiteProfile {
	return unit3DSiteProfile{resolveTypeID: resolveUnit3DSTCTypeID}
}

func resolveUnit3DSTCTypeID(meta api.PreparedMetadata) string {
	typeValue := inferUnit3DType(meta)
	mapping := map[string]string{
		"DISC":   "1",
		"REMUX":  "2",
		"WEBDL":  "4",
		"WEBRIP": "5",
		"HDTV":   "6",
		"ENCODE": "3",
	}
	if meta.TVPack {
		isWeb := typeValue == "WEBDL" || typeValue == "WEBRIP"
		sd := isSDResolution(resolveResolution(meta))
		if sd {
			if isWeb {
				return "14"
			}
			return "17"
		}
		if isWeb {
			return "13"
		}
		return "18"
	}
	return mapping[typeValue]
}
