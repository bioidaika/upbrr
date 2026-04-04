// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package guiapp

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestCandidateAssetPaths(t *testing.T) {
	t.Parallel()

	paths := candidateAssetPaths()

	if !slices.Contains(paths, filepath.Join("gui", "frontend", "dist")) {
		t.Fatalf("missing repo-root asset path: %v", paths)
	}
	if !slices.Contains(paths, filepath.Join("frontend", "dist")) {
		t.Fatalf("missing gui-working-directory asset path: %v", paths)
	}
}
