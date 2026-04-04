// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"

	"github.com/autobrr/upbrr/pkg/api"
)

type stubHandler struct {
	reason string
}

func (h stubHandler) Search(_ context.Context, _ api.PreparedMetadata, tracker string) ([]api.DupeEntry, []string, error) {
	reason := h.reason
	if reason == "" {
		reason = handlerNotImplementedReason(tracker)
	}
	return nil, []string{noteSkip(reason)}, nil
}
