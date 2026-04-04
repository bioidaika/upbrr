// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metadata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyTagOverrides(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "tags.json")
	data := []byte(`{"SubsPlease":{"type":"WEBDL","source":"WEB","in_name":"SubsPlease","personalrelease":"true"},"Pasta":{"type":"WEBRIP"}}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write tags json: %v", err)
	}

	t.Run("in-name override", func(t *testing.T) {
		t.Parallel()
		tag, override, err := ApplyTagOverrides("/media/SubsPlease.Show.mkv", "", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tag != "-SubsPlease" {
			t.Fatalf("expected tag -SubsPlease, got %q", tag)
		}
		if override == nil || override.Type != "WEBDL" || override.Source != "WEB" || !override.PersonalRelease {
			t.Fatalf("unexpected override: %+v", override)
		}
	})

	t.Run("explicit tag override", func(t *testing.T) {
		t.Parallel()
		tag, override, err := ApplyTagOverrides("/media/Pasta.Show.mkv", "-Pasta", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tag != "-Pasta" {
			t.Fatalf("expected tag -Pasta, got %q", tag)
		}
		if override == nil || override.Type != "WEBRIP" {
			t.Fatalf("unexpected override: %+v", override)
		}
	})
}
