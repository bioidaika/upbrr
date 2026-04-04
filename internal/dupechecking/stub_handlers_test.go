// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestStubHandlerReturnsSkipNote(t *testing.T) {
	t.Parallel()
	h := stubHandler{}
	_, notes, err := h.Search(context.Background(), api.PreparedMetadata{}, "AR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected a single note")
	}
	if reason, ok := parseSkipReason(notes); !ok || reason == "" {
		t.Fatalf("expected parseable skip reason")
	}
}
