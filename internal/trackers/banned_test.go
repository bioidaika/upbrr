// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package trackers

import (
	"path/filepath"
	"testing"
)

func TestNewBannedGroupCheckerFromDBPath(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	checker := NewBannedGroupChecker(filepath.Join(tempDir, "db.sqlite"))
	if checker == nil {
		t.Fatalf("expected checker, got nil")
	}
	bannedDir := filepath.Join(tempDir, "cache", "banned")
	if checker.basePath != bannedDir {
		t.Fatalf("expected base path %q, got %q", bannedDir, checker.basePath)
	}
}

func TestNewBannedGroupCheckerNoPathUsesDefaultRoot(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	checker := NewBannedGroupChecker(" ")
	if checker == nil {
		t.Fatalf("expected checker")
	}
	expected := filepath.Join(home, ".upbrr", "cache", "banned")
	if checker.basePath != expected {
		t.Fatalf("expected base path %q, got %q", expected, checker.basePath)
	}
}
