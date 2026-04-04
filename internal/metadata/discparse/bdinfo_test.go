// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package discparse

import "testing"

func TestParseBDInfoFiles(t *testing.T) {
	input := "00001.m2ts        00:10:00     1,000,000,000  25.00 Mbps"
	files := ParseBDInfoFiles(input)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].File != "00001.m2ts" {
		t.Fatalf("unexpected file name: %q", files[0].File)
	}
	if files[0].Length != "00:10:00" {
		t.Fatalf("unexpected length: %q", files[0].Length)
	}
}

func TestParseBDInfoSummary(t *testing.T) {
	summary := "Playlist: 00001.MPLS\nDisc Size: 50,000,000,000 bytes\nLength: 01:30:00.000\nVideo: AVC / 30000 kbps / 1920x1080 / 23.976 fps / 16:9 / High / 8 bits / HDR10 / BT.2020\nAudio: English / DTS-HD Master Audio / 5.1 / 48 kHz / 3500 kbps / 24-bit\nSubtitle: English / SDH\nDisc Title: Example\nDisc Label: EXAMPLE"
	files := "00001.m2ts        00:10:00     1,000,000,000  25.00 Mbps"
	info := ParseBDInfoSummary(summary, files, "/disc/BDMV")
	if info.Playlist != "00001" {
		t.Fatalf("unexpected playlist: %q", info.Playlist)
	}
	if len(info.Video) != 1 {
		t.Fatalf("expected 1 video track")
	}
	if len(info.Audio) != 1 {
		t.Fatalf("expected 1 audio track")
	}
	if len(info.Subtitles) != 1 {
		t.Fatalf("expected 1 subtitle")
	}
}

func TestSplitBDInfoReport(t *testing.T) {
	report := "FILES:\n-------------\n00001.m2ts        00:10:00     1,000,000,000\nCHAPTERS:\nQUICK SUMMARY:\nPlaylist: 00001.MPLS\n********************\n[code]\nIGNORE\n[/code]\n[code]\nSUMMARY\nFILES:\n"
	summary, files, ext := SplitBDInfoReport(report)
	if summary == "" {
		t.Fatalf("expected summary")
	}
	if files == "" {
		t.Fatalf("expected files section")
	}
	if ext == "" {
		t.Fatalf("expected ext summary")
	}
}

func TestSplitBDInfoReportExtSummaryUsesSecondCodeMarker(t *testing.T) {
	tests := []struct {
		name   string
		report string
		want   string
	}{
		{
			name: "two code markers",
			report: "QUICK SUMMARY:\nS\n********************\n" +
				"[code]first block\nFILES:\nignore\n[/code]\n" +
				"[code]second block summary\nFILES:\nuse this\n[/code]",
			want: "second block summary",
		},
		{
			name: "three code markers",
			report: "QUICK SUMMARY:\nS\n********************\n" +
				"[code]first block\nFILES:\nignore\n[/code]\n" +
				"[code]second block summary\nFILES:\nuse this\n[/code]\n" +
				"[code]third block\nFILES:\nignore\n[/code]",
			want: "second block summary",
		},
		{
			name: "four code markers",
			report: "QUICK SUMMARY:\nS\n********************\n" +
				"[code]first block\nFILES:\nignore\n[/code]\n" +
				"[code]second block summary\nFILES:\nuse this\n[/code]\n" +
				"[code]third block\nFILES:\nignore\n[/code]\n" +
				"[code]fourth block\nFILES:\nignore\n[/code]",
			want: "second block summary",
		},
		{
			name: "missing closing tag",
			report: "QUICK SUMMARY:\nS\n********************\n" +
				"[code]first block\nFILES:\nignore\n[/code]\n" +
				"[code]second block summary\nFILES:\nuse this\n",
			want: "second block summary",
		},
		{
			name: "extra closing tag",
			report: "QUICK SUMMARY:\nS\n********************\n" +
				"[code]first block\nFILES:\nignore\n[/code]\n" +
				"[code]second block summary[/code][/code]\nFILES:\nuse this\n",
			want: "second block summary[/code][/code]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, got := SplitBDInfoReport(tt.report)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
