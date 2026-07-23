// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nethd

import (
	"context"

	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/pkg/api"
)

type definition struct{}

func New() trackers.Definition {
	return definition{}
}

func (definition) Name() string {
	return trackerName
}

func (definition) Upload(ctx context.Context, req trackers.UploadRequest) (api.UploadSummary, error) {
	return upload(ctx, req)
}

func (definition) BuildUploadDryRun(ctx context.Context, req trackers.UploadRequest) (api.TrackerDryRunEntry, error) {
	return buildUploadDryRun(ctx, req)
}

func (definition) BuildDescription(ctx context.Context, req trackers.DescriptionRequest) (trackers.DescriptionResult, error) {
	assets, err := trackers.ResolveDescriptionAssetsWithPrepared(ctx, req.Tracker, req.Meta, req.Repo, req.Logger, req.Assets)
	if err != nil {
		trackers.LogDescriptionAssetResolutionFailure(req.Logger, req.Tracker, err)
		assets = trackers.DescriptionAssets{}
	}
	description := buildDescription(trackers.UploadRequest{
		Tracker:       req.Tracker,
		Meta:          req.Meta,
		TrackerConfig: req.TrackerConfig,
		AppConfig:     req.AppConfig,
		Logger:        req.Logger,
		Repo:          req.Repo,
		Assets:        req.Assets,
	}, assets)
	return trackers.DescriptionResult{Group: "nethd", Description: description}, nil
}
