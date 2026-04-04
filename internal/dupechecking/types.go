// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"net/http"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/trackerdata"
	"github.com/autobrr/upbrr/pkg/api"
)

type searchHandler interface {
	Search(ctx context.Context, meta api.PreparedMetadata, tracker string) ([]api.DupeEntry, []string, error)
}

type handlerDeps struct {
	cfg     config.Config
	logger  api.Logger
	http    *http.Client
	tracker *trackerdata.Client
}
