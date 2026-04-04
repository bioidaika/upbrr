// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

type Image struct {
	ImgURL string
	RawURL string
	WebURL string
	Host   string
}

type Note struct {
	Kind    string
	Message string
}

type Report struct {
	Description string
	Images      []Image
	Notes       []Note
}
