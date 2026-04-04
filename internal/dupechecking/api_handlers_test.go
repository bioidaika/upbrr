// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dupechecking

import (
	"context"
	"testing"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestAPIHandlersMissingCredentialsSkip(t *testing.T) {
	t.Parallel()
	meta := api.PreparedMetadata{SourcePath: "x", ExternalIDs: api.ExternalIDs{TMDBID: 1, IMDBID: 1}}
	cases := []struct {
		name    string
		handler searchHandler
	}{
		{name: "ANT", handler: antHandler{cfg: config.Config{}}},
		{name: "AR", handler: arHandler{cfg: config.Config{}}},
		{name: "BHD", handler: bhdHandler{cfg: config.Config{}}},
		{name: "HDB", handler: hdbHandler{cfg: config.Config{}}},
		{name: "PTP", handler: ptpHandler{cfg: config.Config{}}},
		{name: "DC", handler: dcHandler{cfg: config.Config{}}},
		{name: "GPW", handler: gpwHandler{cfg: config.Config{}}},
		{name: "NBL", handler: nblHandler{cfg: config.Config{}}},
		{name: "MTV", handler: mtvHandler{cfg: config.Config{}}},
		{name: "RTF", handler: rtfHandler{cfg: config.Config{}}},
		{name: "SPD", handler: spdHandler{cfg: config.Config{}}},
		{name: "TL", handler: tlHandler{cfg: config.Config{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, notes, err := tc.handler.Search(context.Background(), meta, tc.name)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := parseSkipReason(notes); !ok {
				t.Fatalf("expected skip reason notes, got %#v", notes)
			}
		})
	}
}

func TestBHDTVHandlerReturnsManualMessage(t *testing.T) {
	t.Parallel()
	entries, notes, err := (bhdtvHandler{}).Search(context.Background(), api.PreparedMetadata{}, "BHDTV")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one manual entry")
	}
	if len(notes) == 0 {
		t.Fatalf("expected notes")
	}
}

func TestTVCHandlerSkipsRestrictedContent(t *testing.T) {
	t.Parallel()
	meta := api.PreparedMetadata{Release: api.ReleaseInfo{Resolution: "2160p"}}
	_, notes, err := (tvcHandler{}).Search(context.Background(), meta, "TVC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := parseSkipReason(notes); !ok {
		t.Fatalf("expected skip reason")
	}
}
