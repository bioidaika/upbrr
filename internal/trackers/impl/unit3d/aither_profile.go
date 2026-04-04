// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/internal/trackers"
)

func siteAITHERProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		applyAdditionalPayload: func(req trackers.UploadRequest, data map[string]string) {
			hdrValue := strings.ToUpper(strings.TrimSpace(req.Meta.HDR))
			hasHDR10P := strings.Contains(hdrValue, "HDR10+")

			data["dv"] = boolFlag(strings.Contains(hdrValue, "DV"))
			if hasHDR10P {
				data["hdr10p"] = "1"
				return
			}
			if strings.Contains(hdrValue, "HDR") || strings.Contains(hdrValue, "HLG") {
				data["hdr"] = "1"
			}
		},
	}
}
