// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"net/http"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/pkg/api"
)

type czHandler struct {
	cfg    config.Config
	http   *http.Client
	logger api.Logger
}

func (h czHandler) Search(ctx context.Context, meta api.PreparedMetadata, tracker string) ([]api.DupeEntry, []string, error) {
	return azNetworkHandler(h).Search(ctx, meta, tracker)
}
