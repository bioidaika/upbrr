// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package trackers

import (
	"context"
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestEvaluateRulesAZRedirectsEnglishTerritories(t *testing.T) {
	t.Parallel()

	failures := EvaluateRules(context.Background(), "AZ", api.PreparedMetadata{
		ExternalIDs: api.ExternalIDs{Category: "MOVIE"},
		ExternalMetadata: api.ExternalMetadata{
			TMDB: &api.TMDBMetadata{OriginCountry: []string{"US"}},
		},
	}, api.NopLogger{})
	if len(failures) == 0 {
		t.Fatal("expected AZ rule failure")
	}
}

func TestEvaluateRulesCZRejectsAsianContent(t *testing.T) {
	t.Parallel()

	failures := EvaluateRules(context.Background(), "CZ", api.PreparedMetadata{
		ExternalIDs: api.ExternalIDs{Category: "MOVIE"},
		ExternalMetadata: api.ExternalMetadata{
			TMDB: &api.TMDBMetadata{OriginCountry: []string{"JP"}},
		},
	}, api.NopLogger{})
	if len(failures) == 0 {
		t.Fatal("expected CZ rule failure")
	}
}

func TestEvaluateRulesPHDRejectsSDAndBlockedGroup(t *testing.T) {
	t.Parallel()

	failures := EvaluateRules(context.Background(), "PHD", api.PreparedMetadata{
		ExternalIDs: api.ExternalIDs{Category: "MOVIE"},
		Release:     api.ReleaseInfo{Resolution: "480p"},
		Container:   "avi",
		Tag:         "-RARBG",
	}, api.NopLogger{})
	if len(failures) < 2 {
		t.Fatalf("expected multiple PHD failures, got %v", failures)
	}
}
