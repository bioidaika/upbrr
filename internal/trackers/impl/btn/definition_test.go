// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package btn

import "testing"

func TestDefinitionName(t *testing.T) {
	t.Parallel()

	def := New()
	if def.Name() != "BTN" {
		t.Fatalf("expected BTN, got %q", def.Name())
	}
}

func TestApplyBTNNameMapping(t *testing.T) {
	t.Parallel()

	name := "Example.Show.S01E01.1080p.WEB-DL.x265-GRP"
	mapped := applyBTNNameMapping(name, "H.265", "WEB-DL")
	if mapped == "" {
		t.Fatalf("expected mapped name")
	}
	if mapped != "Example.Show.S01E01.1080p.WEB-DL.H.265-GRP" {
		t.Fatalf("unexpected mapped name: %s", mapped)
	}
}
