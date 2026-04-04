// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package api

import "time"

type UploadedImageLink struct {
	SourcePath string
	ImagePath  string
	Host       string
	UsageScope string
	ImgURL     string
	RawURL     string
	WebURL     string
	SizeBytes  int64
	UploadedAt time.Time
}
