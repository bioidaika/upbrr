// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package screenshots

import (
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestBuildScreenshotSelections(t *testing.T) {
	meta := api.PreparedMetadata{}
	selections := buildScreenshotSelections(4, 600, 24, meta)
	if len(selections) != 4 {
		t.Fatalf("expected 4 selections, got %d", len(selections))
	}
	prev := -1.0
	for _, sel := range selections {
		if sel.TimestampSeconds <= 0 {
			t.Fatalf("expected positive timestamp, got %f", sel.TimestampSeconds)
		}
		if sel.TimestampSeconds <= prev {
			t.Fatalf("timestamps not increasing: %f <= %f", sel.TimestampSeconds, prev)
		}
		prev = sel.TimestampSeconds
	}
}

func TestSanitizeFilename(t *testing.T) {
	got := sanitizeFilename("My:File/Name")
	if got == "" || got == "My:File/Name" {
		t.Fatalf("expected sanitized filename, got %q", got)
	}
}

func TestBuildManualFrameSelections(t *testing.T) {
	selections := buildManualFrameSelections([]int{240, 480, 720}, 24)
	if len(selections) != 3 {
		t.Fatalf("expected 3 selections, got %d", len(selections))
	}
	for idx, selection := range selections {
		if selection.Index != idx {
			t.Fatalf("expected index %d, got %d", idx, selection.Index)
		}
		if selection.Frame != (idx+1)*240 {
			t.Fatalf("unexpected frame at %d: %#v", idx, selection)
		}
		if selection.Source != "manual" {
			t.Fatalf("expected manual source, got %#v", selection)
		}
	}
	if selections[1].TimestampSeconds != 20 {
		t.Fatalf("expected second timestamp 20, got %f", selections[1].TimestampSeconds)
	}
}
