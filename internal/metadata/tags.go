// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metadata

import "strings"

func DetectTag(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	group := strings.TrimSpace(ParseReleaseInfo(trimmed).Group)
	if group == "" {
		return ""
	}

	return "-" + group
}
