// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ar

import (
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestResolveARNameAddsNoGRP(t *testing.T) {
	t.Parallel()

	got := resolveARName(api.PreparedMetadata{
		SourcePath: "C:/data/My Movie (2024).mkv",
		Release:    api.ReleaseInfo{Title: "My Movie", Year: 2024},
	})
	if got != "My.Movie.2024-NoGRP" {
		t.Fatalf("unexpected AR name %q", got)
	}
}

func TestResolveARNameUsesSceneName(t *testing.T) {
	t.Parallel()

	got := resolveARName(api.PreparedMetadata{
		Scene:     true,
		SceneName: "Scene.Release-GRP",
		Tag:       "-GRP",
	})
	if got != "Scene.Release-GRP" {
		t.Fatalf("expected scene name, got %q", got)
	}
}
