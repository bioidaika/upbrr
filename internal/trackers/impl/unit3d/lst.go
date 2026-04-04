// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/internal/trackers"
)

func siteLSTProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		applyAdditionalPayload: func(req trackers.UploadRequest, data map[string]string) {
			data["draft_queue_opt_in"] = boolFlag(req.TrackerConfig.Draft)
			if editionID, ok := resolveLSTEditionID(req.Meta.Edition); ok {
				data["edition_id"] = editionID
			}
		},
	}
}

func resolveLSTEditionID(edition string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(edition))
	normalized = strings.ReplaceAll(normalized, "’", "'")
	mapping := map[string]string{
		"collector's edition": "1",
		"director's cut":      "2",
		"extended cut":        "3",
		"extended uncut":      "4",
		"extended unrated":    "5",
		"limited edition":     "6",
		"special edition":     "7",
		"theatrical cut":      "8",
		"uncut":               "9",
		"unrated":             "10",
		"x cut":               "11",
		"alternative cut":     "12",
		"other":               "0",
	}
	value, ok := mapping[normalized]
	return value, ok
}
