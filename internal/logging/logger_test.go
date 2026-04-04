// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logging

import "testing"

func TestResolveEffectiveLevel(t *testing.T) {
	t.Parallel()

	if got := ResolveEffectiveLevel("info", "", false); got != "info" {
		t.Fatalf("expected configured level info, got %q", got)
	}
	if got := ResolveEffectiveLevel("info", "", true); got != "debug" {
		t.Fatalf("expected debug fallback for debug runs, got %q", got)
	}
	if got := ResolveEffectiveLevel("info", "trace", true); got != "trace" {
		t.Fatalf("expected explicit override trace, got %q", got)
	}
}
