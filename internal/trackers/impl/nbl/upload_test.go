// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nbl

import "testing"

func TestExtractUploadLinkAndIDFromRootLink(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"link": "https://nebulance.io/torrents.php?id=12345",
	}

	link, id := extractUploadLinkAndID(payload)
	if link != "https://nebulance.io/torrents.php?id=12345" {
		t.Fatalf("expected upload link from root, got %q", link)
	}
	if id != "12345" {
		t.Fatalf("expected torrent id 12345, got %q", id)
	}
}

func TestExtractUploadLinkAndIDFromResultLink(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"result": map[string]any{
			"link": "https://nebulance.io/torrents.php?id=67890",
		},
	}

	link, id := extractUploadLinkAndID(payload)
	if link != "https://nebulance.io/torrents.php?id=67890" {
		t.Fatalf("expected upload link from nested result, got %q", link)
	}
	if id != "67890" {
		t.Fatalf("expected torrent id 67890, got %q", id)
	}
}

func TestExtractUploadLinkAndIDEmptyWhenMissingLink(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"status": "ok"}

	link, id := extractUploadLinkAndID(payload)
	if link != "" {
		t.Fatalf("expected empty upload link when missing, got %q", link)
	}
	if id != "" {
		t.Fatalf("expected empty torrent id when link missing, got %q", id)
	}
}
