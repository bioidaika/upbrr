// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"

	"github.com/autobrr/upbrr/pkg/api"
)

type bhdtvHandler struct{}

func (h bhdtvHandler) Search(_ context.Context, _ api.PreparedMetadata, _ string) ([]api.DupeEntry, []string, error) {
	return stringsToDupeEntries([]string{"Dupes must be checked Manually"}), []string{"BHDTV API dupe search is not available; manual check required"}, nil
}
