// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package guiapp

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

func resolveAssets(override fs.FS) (fs.FS, error) {
	return ResolveAssets(override)
}

func ResolveAssets(override fs.FS) (fs.FS, error) {
	if override != nil {
		return override, nil
	}
	if embeddedRoot, err := fs.Sub(embeddedAssets, "assets"); err == nil {
		if hasIndex(embeddedRoot) {
			return embeddedRoot, nil
		}
	}
	if hasIndex(embeddedAssets) {
		return embeddedAssets, nil
	}
	for _, path := range candidateAssetPaths() {
		diskAssets := os.DirFS(path)
		if hasIndex(diskAssets) {
			return diskAssets, nil
		}
	}
	return nil, errors.New("gui assets not found: build gui/frontend and retry")
}

func hasIndex(assets fs.FS) bool {
	if assets == nil {
		return false
	}
	_, err := fs.Stat(assets, "index.html")
	return err == nil
}

func candidateAssetPaths() []string {
	return []string{
		filepath.Join("gui", "frontend", "dist"),
		filepath.Join("frontend", "dist"),
	}
}
