// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestVMFUsesSharedUnit3DResolutionMapping(t *testing.T) {
	t.Parallel()

	meta := api.PreparedMetadata{Release: api.ReleaseInfo{Resolution: "1440p"}}
	if got := resolveUnit3DResolutionIDForTracker("VMF", meta); got != "3" {
		t.Fatalf("expected shared Unit3D resolution_id=3 for VMF 1440p, got %q", got)
	}
}

func TestVMFUsesSharedUnit3DTypeMapping(t *testing.T) {
	t.Parallel()

	got, err := resolveUnit3DTypeIDForTracker("VMF", api.PreparedMetadata{Type: "DVDRIP"})
	if err != nil {
		t.Fatalf("expected shared Unit3D DVDRIP mapping for VMF, got error: %v", err)
	}
	if got != "3" {
		t.Fatalf("expected shared Unit3D type_id=3 for VMF DVDRIP, got %q", got)
	}
}
