// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metadata

import (
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestExtractDVDMediaInfoFromVOBJSON(t *testing.T) {
	meta := api.PreparedMetadata{
		DiscType:            "DVD",
		SourcePath:          `/releases/Movie.DVD`,
		DVDVOBMediaInfoJSON: `{"media":{"track":[{"@type":"General"},{"@type":"Video","Width":"720","Height":"576","FrameRate":"25.000","ScanType":"Interlaced"}]}}`,
	}

	info := extractDVDMediaInfo(meta)
	if info.Width != 720 || info.Height != 576 {
		t.Fatalf("expected 720x576, got %dx%d", info.Width, info.Height)
	}
	if info.ScanType != "i" {
		t.Fatalf("expected scan i, got %q", info.ScanType)
	}
	if info.Resolution != "576i" {
		t.Fatalf("expected 576i, got %q", info.Resolution)
	}
}

func TestExtractDVDMediaInfoFallsBackToText(t *testing.T) {
	meta := api.PreparedMetadata{
		DiscType:            "DVD",
		SourcePath:          `/releases/Movie.DVD`,
		DVDVOBMediaInfoJSON: `{"media":{"track":[{"@type":"General"},{"@type":"Video"}]}}`,
		DVDVOBMediaInfoText: "Width : 720\nHeight : 480\nFrame rate : 29.970\nScan type : Interlaced\n",
	}

	info := extractDVDMediaInfo(meta)
	if info.Width != 720 || info.Height != 480 {
		t.Fatalf("expected 720x480, got %dx%d", info.Width, info.Height)
	}
	if info.FrameRate != "29.970" {
		t.Fatalf("expected frame rate from text, got %q", info.FrameRate)
	}
	if info.ScanType != "i" {
		t.Fatalf("expected scan i, got %q", info.ScanType)
	}
	if info.Resolution != "480i" {
		t.Fatalf("expected 480i, got %q", info.Resolution)
	}
}

func TestExtractDVDMediaInfoUsesInterlacedHintFromSourcePath(t *testing.T) {
	meta := api.PreparedMetadata{
		DiscType:            "DVD",
		SourcePath:          `/releases/Movie.1080i.DVD`,
		DVDVOBMediaInfoJSON: `{"media":{"track":[{"@type":"General"},{"@type":"Video","Width":"1920","Height":"1080","FrameRate":"25.000"}]}}`,
	}

	info := extractDVDMediaInfo(meta)
	if info.ScanType != "i" {
		t.Fatalf("expected scan i from source hint, got %q", info.ScanType)
	}
	if info.Resolution != "1080i" {
		t.Fatalf("expected 1080i, got %q", info.Resolution)
	}
}

func TestExtractDVDMediaInfoDefaultsUnknownScanToProgressive(t *testing.T) {
	meta := api.PreparedMetadata{
		DiscType:            "DVD",
		SourcePath:          `/releases/Movie.DVD`,
		DVDVOBMediaInfoJSON: `{"media":{"track":[{"@type":"General"},{"@type":"Video","Width":"720","Height":"576","FrameRate":"25.000"}]}}`,
	}

	info := extractDVDMediaInfo(meta)
	if info.ScanType != "p" {
		t.Fatalf("expected unknown scan to default to p, got %q", info.ScanType)
	}
	if info.Resolution != "576p" {
		t.Fatalf("expected 576p, got %q", info.Resolution)
	}
}
