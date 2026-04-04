// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

type tvcHandler struct{}

func (h tvcHandler) Search(_ context.Context, meta api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	res := strings.ToLower(strings.TrimSpace(meta.Release.Resolution))
	if strings.Contains(res, "2160") || strings.EqualFold(meta.Type, "REMUX") || strings.TrimSpace(meta.DiscType) != "" {
		return nil, []string{noteSkip("TVC disallows UHD/disc/remux content")}, nil
	}
	return nil, []string{noteSkip("TVC dupe search currently unavailable; manual check required")}, nil
}
