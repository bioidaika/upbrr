// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package api

import "context"

type MetadataProgressUpdate struct {
	Path      string `json:"path"`
	Phase     string `json:"phase"`
	Message   string `json:"message"`
	Status    string `json:"status"`
	Level     string `json:"level"`
	Timestamp string `json:"timestamp"`
}

type MetadataProgressReporter func(update MetadataProgressUpdate)

type metadataProgressReporterKey struct{}

func WithMetadataProgressReporter(ctx context.Context, reporter MetadataProgressReporter) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, metadataProgressReporterKey{}, reporter)
}

func EmitMetadataProgress(ctx context.Context, update MetadataProgressUpdate) {
	if ctx == nil {
		return
	}
	reporter, _ := ctx.Value(metadataProgressReporterKey{}).(MetadataProgressReporter)
	if reporter == nil {
		return
	}
	reporter(update)
}
