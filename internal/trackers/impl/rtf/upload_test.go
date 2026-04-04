// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package rtf

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestBuildUploadDryRunUsesDescriptionPayloadAndLeavesNFOEmpty(t *testing.T) {
	root := t.TempDir()
	torrentPath := filepath.Join(root, "test.torrent")
	if err := os.WriteFile(torrentPath, []byte("torrent-bytes"), 0o600); err != nil {
		t.Fatalf("write torrent: %v", err)
	}

	entry, err := buildUploadDryRun(context.Background(), trackers.UploadRequest{
		Tracker: "RTF",
		Meta: api.PreparedMetadata{
			TorrentPath:         torrentPath,
			DescriptionOverride: "Custom description",
		},
		TrackerConfig: config.TrackerConfig{APIKey: "token"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.Payload["description"] != "Custom description" {
		t.Fatalf("expected description payload, got %#v", entry.Payload)
	}
	if entry.Payload["descr"] != "Custom description" {
		t.Fatalf("expected descr payload mirror, got %#v", entry.Payload)
	}
	if entry.Payload["nfo"] != "" {
		t.Fatalf("expected empty nfo payload, got %#v", entry.Payload["nfo"])
	}
}
