// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package trackers

import (
	"context"

	"github.com/autobrr/upbrr/pkg/api"
)

type UploadDryRunBuilder interface {
	BuildUploadDryRun(ctx context.Context, req UploadRequest) (api.TrackerDryRunEntry, error)
}
