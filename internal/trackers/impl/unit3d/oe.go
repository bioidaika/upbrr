// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteOEProfile() unit3DSiteProfile {
	return unit3DSiteProfile{resolveTypeID: resolveUnit3DOETypeID}
}

func resolveUnit3DOETypeID(meta api.PreparedMetadata) string {
	typeValue := inferUnit3DType(meta)
	if typeValue == "DVDRIP" {
		typeValue = "ENCODE"
	}

	switch typeValue {
	case "DISC":
		return "19"
	case "REMUX":
		return "20"
	case "WEBDL":
		return "21"
	case "WEBRIP", "ENCODE":
		switch normalizeUnit3DVideoCodec(meta.VideoCodec) {
		case "HEVC":
			return "10"
		case "AV1":
			return "14"
		case "AVC":
			return "15"
		default:
			return "16"
		}
	default:
		return "16"
	}
}

func normalizeUnit3DVideoCodec(value string) string {
	codec := strings.ToUpper(strings.TrimSpace(value))
	switch {
	case strings.Contains(codec, "AV1"):
		return "AV1"
	case strings.Contains(codec, "HEVC") || strings.Contains(codec, "H.265") || strings.Contains(codec, "H265") || strings.Contains(codec, "X265"):
		return "HEVC"
	case strings.Contains(codec, "AVC") || strings.Contains(codec, "H.264") || strings.Contains(codec, "H264") || strings.Contains(codec, "X264"):
		return "AVC"
	default:
		return ""
	}
}
