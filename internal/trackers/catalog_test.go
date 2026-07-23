// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package trackers

import (
	"slices"
	"testing"
)

func TestNETHDTrackerCatalog(t *testing.T) {
	t.Parallel()

	if got := TrackerKind(" nethd "); got != KindNonUnit3D {
		t.Fatalf("TrackerKind(NETHD) = %q, want %q", got, KindNonUnit3D)
	}
	if !IsKnownTracker("NETHD") {
		t.Fatal("NETHD should be a known tracker")
	}
	if !IsNonUnit3DTracker("nethd") {
		t.Fatal("NETHD should be a non-Unit3D tracker")
	}
	if IsUnit3DTracker("NETHD") {
		t.Fatal("NETHD should not be a Unit3D tracker")
	}
	if !slices.Contains(NonUnit3DTrackers(), "NETHD") {
		t.Fatal("non-Unit3D tracker list should include NETHD")
	}
	if skipsModifiedReleaseCheck("NETHD") {
		t.Fatal("NETHD should retain the modified-release check")
	}
}
