// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metadata

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestApplyMediaInfoIDsFromJSON(t *testing.T) {
	base := t.TempDir()
	jsonPath := filepath.Join(base, "MediaInfo.json")
	payload := `{"media":{"track":[{"@type":"General","extra":{"IMDB":"tt34581826","TMDB":"movie/1386827","TVDB2":"movies/361746"}}]}}`
	if err := os.WriteFile(jsonPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	svc := &Service{}
	meta, err := svc.ApplyMediaInfoIDs(context.Background(), api.PreparedMetadata{MediaInfoJSONPath: jsonPath})
	if err != nil {
		t.Fatalf("apply mediainfo ids: %v", err)
	}
	if meta.MediaInfoTMDBID != 1386827 {
		t.Fatalf("expected tmdb 1386827, got %d", meta.MediaInfoTMDBID)
	}
	if meta.MediaInfoIMDBID != 34581826 {
		t.Fatalf("expected imdb 34581826, got %d", meta.MediaInfoIMDBID)
	}
	if meta.MediaInfoTVDBID != 361746 {
		t.Fatalf("expected tvdb 361746, got %d", meta.MediaInfoTVDBID)
	}
	if meta.MediaInfoCategory != "movie" {
		t.Fatalf("expected category movie, got %q", meta.MediaInfoCategory)
	}
}

func TestApplyMediaInfoIDsTextTVDBPriority(t *testing.T) {
	base := t.TempDir()
	textPath := filepath.Join(base, "mediainfo.txt")
	payload := "Writing library                          : libebml v1.4.5 + libmatroska v1.7.1\nIMDB                                     : tt32328599\nTMDB                                     : tv/256953\nTVDB                                     : 451293\nTVDB2                                    : episodes/11521397\n"
	if err := os.WriteFile(textPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write text: %v", err)
	}

	svc := &Service{}
	meta, err := svc.ApplyMediaInfoIDs(context.Background(), api.PreparedMetadata{MediaInfoTextPath: textPath})
	if err != nil {
		t.Fatalf("apply mediainfo ids: %v", err)
	}
	if meta.MediaInfoTVDBID != 451293 {
		t.Fatalf("expected tvdb 451293, got %d", meta.MediaInfoTVDBID)
	}
}

func TestApplyMediaInfoIDsMismatch(t *testing.T) {
	base := t.TempDir()
	jsonPath := filepath.Join(base, "MediaInfo.json")
	payload := `{"media":{"track":[{"@type":"General","extra":{"TMDB":"movie/123"}}]}}`
	if err := os.WriteFile(jsonPath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	svc := &Service{}
	meta, err := svc.ApplyMediaInfoIDs(context.Background(), api.PreparedMetadata{
		MediaInfoJSONPath: jsonPath,
		TrackerData:       []api.TrackerMetadata{{TMDBID: 999}},
	})
	if err != nil {
		t.Fatalf("apply mediainfo ids: %v", err)
	}
	if meta.MismatchedMediaInfoTMDBID != 123 {
		t.Fatalf("expected mismatched tmdb 123, got %d", meta.MismatchedMediaInfoTMDBID)
	}
}
