// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestBuildVMFNameAddsTokenAwareVietnameseTag(t *testing.T) {
	t.Parallel()

	meta := api.PreparedMetadata{
		AudioLanguages: []string{"Vietnamese"},
		Release: api.ReleaseInfo{
			Resolution: "1080p",
		},
	}

	name := "Example Movie 2026 1080p WEB-DL DDP5.1 H.264-GRP"
	want := "Example Movie 2026 ViE 1080p WEB-DL DDP5.1 H.264-GRP"
	if got := buildVMFName(name, meta); got != want {
		t.Fatalf("expected VMF name %q, got %q", want, got)
	}
}

func TestBuildVMFNameDoesNotTreatTitleWordVieAsTag(t *testing.T) {
	t.Parallel()

	meta := api.PreparedMetadata{
		AudioLanguages: []string{"Vietnamese"},
		Release: api.ReleaseInfo{
			Resolution: "1080p",
		},
	}

	name := "Example Vie Story 2026 1080p WEB-DL DDP5.1 H.264-GRP"
	want := "Example Vie Story 2026 ViE 1080p WEB-DL DDP5.1 H.264-GRP"
	if got := buildVMFName(name, meta); got != want {
		t.Fatalf("expected VMF name %q, got %q", want, got)
	}
}

func TestBuildVMFNameDoesNotTreatUppercaseTitleWordVIEAsTag(t *testing.T) {
	t.Parallel()

	meta := api.PreparedMetadata{
		AudioLanguages: []string{"Vietnamese"},
		Release: api.ReleaseInfo{
			Resolution: "1080p",
		},
	}

	name := "EXAMPLE.VIE.STORY.2026.1080p.WEB-DL.DDP5.1.H.264-GRP"
	want := "EXAMPLE.VIE.STORY.2026.ViE.1080p.WEB-DL.DDP5.1.H.264-GRP"
	if got := buildVMFName(name, meta); got != want {
		t.Fatalf("expected VMF name %q, got %q", want, got)
	}
}

func TestBuildVMFNamePreservesDotSeparatedConvention(t *testing.T) {
	t.Parallel()

	meta := api.PreparedMetadata{
		AudioTitles: []string{"Vietnamese Lồng Tiếng"},
		Release: api.ReleaseInfo{
			Resolution: "1080p",
		},
	}

	name := "Example.Movie.2026.1080p.NF.WEB-DL.DDP5.1.H.264-GRP"
	want := "Example.Movie.2026.ViE.DUB.1080p.NF.WEB-DL.DDP5.1.H.264-GRP"
	if got := buildVMFName(name, meta); got != want {
		t.Fatalf("expected VMF name %q, got %q", want, got)
	}
}

func TestBuildVMFNameFallsBackToResolutionInName(t *testing.T) {
	t.Parallel()

	meta := api.PreparedMetadata{AudioLanguages: []string{"Vietnamese"}}
	name := "Example Movie 2025 2160p WEB-DL DDP5.1 H.265-GRP"
	want := "Example Movie 2025 ViE 2160p WEB-DL DDP5.1 H.265-GRP"
	if got := buildVMFName(name, meta); got != want {
		t.Fatalf("expected VMF name %q, got %q", want, got)
	}
}

func TestBuildVMFNamePlacesLegacyResolutionAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		releaseName string
		want        string
	}{
		{
			name:        "dot separated 4K",
			releaseName: "Example.Movie.2026.4K.WEB-DL.DDP5.1.H.265-GRP",
			want:        "Example.Movie.2026.ViE.4K.WEB-DL.DDP5.1.H.265-GRP",
		},
		{
			name:        "space separated 8K",
			releaseName: "Example Movie 2026 8K WEB-DL DDP5.1 H.265-GRP",
			want:        "Example Movie 2026 ViE 8K WEB-DL DDP5.1 H.265-GRP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			meta := api.PreparedMetadata{AudioLanguages: []string{"Vietnamese"}}
			if got := buildVMFName(tt.releaseName, meta); got != tt.want {
				t.Fatalf("expected VMF name %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildVMFNameFallsBackToSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		releaseName string
		source      string
		want        string
	}{
		{
			name:        "space separated",
			releaseName: "Example Movie 2025 WEB-DL DDP5.1 H.264-GRP",
			source:      "WEB-DL",
			want:        "Example Movie 2025 ViE WEB-DL DDP5.1 H.264-GRP",
		},
		{
			name:        "dot separated",
			releaseName: "Example.Movie.2025.BluRay.REMUX.DTS-HD.MA.5.1-GRP",
			source:      "BluRay",
			want:        "Example.Movie.2025.ViE.BluRay.REMUX.DTS-HD.MA.5.1-GRP",
		},
		{
			name:        "source word also appears in title",
			releaseName: "Example Web Story 2026 WEB-DL DDP5.1 H.264-GRP",
			source:      "WEB",
			want:        "Example Web Story 2026 ViE WEB-DL DDP5.1 H.264-GRP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			meta := api.PreparedMetadata{
				AudioLanguages: []string{"Vietnamese"},
				Release: api.ReleaseInfo{
					Source: tt.source,
				},
			}
			if got := buildVMFName(tt.releaseName, meta); got != tt.want {
				t.Fatalf("expected VMF name %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildVMFNameFallsBackBeforeReleaseGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		releaseName string
		group       string
		tag         string
		want        string
	}{
		{
			name:        "space separated",
			releaseName: "Example Documentary 2025-GRP",
			group:       "GRP",
			want:        "Example Documentary 2025 ViE-GRP",
		},
		{
			name:        "dot separated",
			releaseName: "Example.Documentary.2025-GRP",
			group:       "GRP",
			want:        "Example.Documentary.2025.ViE-GRP",
		},
		{
			name:        "no-group suffix",
			releaseName: "Example.Documentary.2025-NoGroup",
			group:       "NoGroup",
			want:        "Example.Documentary.2025.ViE-NoGroup",
		},
		{
			name:        "metadata tag fallback",
			releaseName: "Example.Documentary.2025-GRP",
			tag:         " -GRP ",
			want:        "Example.Documentary.2025.ViE-GRP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			meta := api.PreparedMetadata{
				AudioLanguages: []string{"Vietnamese"},
				Tag:            tt.tag,
				Release: api.ReleaseInfo{
					Group: tt.group,
				},
			}
			if got := buildVMFName(tt.releaseName, meta); got != tt.want {
				t.Fatalf("expected VMF name %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildVMFNameReconcilesExistingTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		releaseName    string
		audioTitles    []string
		audioLanguages []string
		want           string
	}{
		{
			name:           "keeps existing ViE",
			releaseName:    "Example Movie 2025 ViE 1080p WEB-DL-GRP",
			audioLanguages: []string{"Vietnamese"},
			want:           "Example Movie 2025 ViE 1080p WEB-DL-GRP",
		},
		{
			name:        "upgrades bare ViE to dub",
			releaseName: "Example Movie 2025 ViE 1080p WEB-DL-GRP",
			audioTitles: []string{"Lồng Tiếng"},
			want:        "Example Movie 2025 ViE DUB 1080p WEB-DL-GRP",
		},
		{
			name:           "does not downgrade dub",
			releaseName:    "Example Movie 2025 ViE DUB 1080p WEB-DL-GRP",
			audioLanguages: []string{"Vietnamese"},
			want:           "Example Movie 2025 ViE DUB 1080p WEB-DL-GRP",
		},
		{
			name:        "recognizes dot separated dub",
			releaseName: "Example.Movie.2025.ViE.DUB.1080p.WEB-DL-GRP",
			audioTitles: []string{"VNLT"},
			want:        "Example.Movie.2025.ViE.DUB.1080p.WEB-DL-GRP",
		},
		{
			name:        "upgrades dot separated ViE",
			releaseName: "Example.Movie.2025.ViE.1080p.WEB-DL-GRP",
			audioTitles: []string{"Lồng Tiếng"},
			want:        "Example.Movie.2025.ViE.DUB.1080p.WEB-DL-GRP",
		},
		{
			name:           "relocates old trailing ViE output",
			releaseName:    "Example Movie 2025 1080p WEB-DL-GRP ViE",
			audioLanguages: []string{"Vietnamese"},
			want:           "Example Movie 2025 ViE 1080p WEB-DL-GRP",
		},
		{
			name:           "relocates trailing dot dub without downgrade",
			releaseName:    "Example.Movie.2025.1080p.WEB-DL-GRP.ViE.DUB",
			audioLanguages: []string{"Vietnamese"},
			want:           "Example.Movie.2025.ViE.DUB.1080p.WEB-DL-GRP",
		},
		{
			name:           "deduplicates conflicting tags",
			releaseName:    "Example Movie 2025 ViE ViE DUB 1080p WEB-DL-GRP ViE",
			audioLanguages: []string{"Vietnamese"},
			want:           "Example Movie 2025 ViE DUB 1080p WEB-DL-GRP",
		},
		{
			name:        "upgrades and relocates old trailing ViE",
			releaseName: "Example Movie 2025 1080p WEB-DL-GRP ViE",
			audioTitles: []string{"VNLT"},
			want:        "Example Movie 2025 ViE DUB 1080p WEB-DL-GRP",
		},
		{
			name:        "preserves existing dub without audio metadata",
			releaseName: "Example Movie 2025 1080p WEB-DL-GRP ViE DUB",
			want:        "Example Movie 2025 ViE DUB 1080p WEB-DL-GRP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			meta := api.PreparedMetadata{
				AudioTitles:    tt.audioTitles,
				AudioLanguages: tt.audioLanguages,
				Release: api.ReleaseInfo{
					Resolution: "1080p",
				},
			}
			got := buildVMFName(tt.releaseName, meta)
			if got != tt.want {
				t.Fatalf("expected VMF name %q, got %q", tt.want, got)
			}
			vieTags := 0
			for _, token := range tokenizeVMFName(got) {
				if token.value == "ViE" {
					vieTags++
				}
			}
			if vieTags != 1 {
				t.Fatalf("expected exactly one VMF ViE tag, got %q", got)
			}
		})
	}
}

func TestBuildVMFNameLeavesNonVietnameseReleaseUnchanged(t *testing.T) {
	t.Parallel()

	name := "Example.Movie.2025.1080p.WEB-DL-GRP"
	meta := api.PreparedMetadata{
		AudioLanguages: []string{"English"},
		Release: api.ReleaseInfo{
			Resolution: "1080p",
		},
	}
	if got := buildVMFName(name, meta); got != name {
		t.Fatalf("expected non-Vietnamese VMF name unchanged, got %q", got)
	}
}
