// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metadata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/autobrr/upbrr/internal/config"
	internalerrors "github.com/autobrr/upbrr/internal/errors"
	"github.com/autobrr/upbrr/internal/metadata/mediainfo"
	"github.com/autobrr/upbrr/internal/services/db"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestPrepare(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	path := filepath.Join(base, "example")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	videoPath := filepath.Join(path, "example.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video failed: %v", err)
	}

	repo := &stubRepo{existing: db.FileMetadata{Path: path, InfoHash: "hash"}}
	cfg := config.Config{MainSettings: config.MainSettingsConfig{DBPath: filepath.Join(base, "db.sqlite")}}
	service := NewService(repo, WithMediaInfoExporter(&stubMediaInfo{}), WithSceneDetector(stubSceneDetector{}), WithConfig(cfg))

	meta, err := service.Prepare(context.Background(), api.Request{
		Paths:          []string{path},
		Mode:           api.ModeCLI,
		Trackers:       []string{"blu", "bhd"},
		Options:        api.UploadOptions{Debug: true, Screens: 3},
		TrackersRemove: []string{"bhd"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if meta.SourcePath != path {
		t.Fatalf("unexpected source path: %s", meta.SourcePath)
	}
	if len(meta.Paths) != 1 {
		t.Fatalf("unexpected paths length: %d", len(meta.Paths))
	}
	if meta.Mode != api.ModeCLI {
		t.Fatalf("unexpected mode: %s", meta.Mode)
	}
	if len(meta.Trackers) != 2 {
		t.Fatalf("unexpected trackers length: %d", len(meta.Trackers))
	}
	if !meta.Options.Debug || meta.Options.Screens != 3 {
		t.Fatalf("unexpected options: %+v", meta.Options)
	}
	if len(meta.TrackersRemove) != 1 {
		t.Fatalf("unexpected trackers-remove length: %d", len(meta.TrackersRemove))
	}
	if len(meta.Paths) != 1 || meta.Paths[0] != path {
		t.Fatalf("unexpected paths: %v", meta.Paths)
	}
	if meta.StoredInfoHash != "hash" {
		t.Fatalf("unexpected stored info hash: %s", meta.StoredInfoHash)
	}
	if !meta.StoredDataFresh {
		t.Fatalf("expected stored metadata marked fresh")
	}
	if repo.saved.InfoHash != "hash" {
		t.Fatalf("expected persisted info hash, got %s", repo.saved.InfoHash)
	}
	if repo.saved.Path != path {
		t.Fatalf("expected repo save path, got %q", repo.saved.Path)
	}

	_, err = service.Prepare(context.Background(), api.Request{})
	if !errors.Is(err, internalerrors.ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got: %v", err)
	}
}

func TestPrepareAppliesTorrentOverrides(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	path := filepath.Join(base, "example")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	videoPath := filepath.Join(path, "example.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video failed: %v", err)
	}

	repo := &stubRepo{}
	cfg := config.Config{MainSettings: config.MainSettingsConfig{DBPath: filepath.Join(base, "db.sqlite")}}
	service := NewService(repo, WithMediaInfoExporter(&stubMediaInfo{}), WithSceneDetector(stubSceneDetector{}), WithConfig(cfg))
	infoHash := "abcdef0123456789abcdef0123456789abcdef01"

	meta, err := service.Prepare(context.Background(), api.Request{
		Paths: []string{path},
		Mode:  api.ModeCLI,
		TorrentOverrides: api.TorrentOverrides{
			InfoHash: &infoHash,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if meta.InfoHash != infoHash {
		t.Fatalf("expected infohash override, got %q", meta.InfoHash)
	}
}

func TestResolveServiceDarkroom(t *testing.T) {
	t.Parallel()

	service, longName, filename := resolveService(api.PreparedMetadata{
		SourcePath: `/releases/Example.Movie.2025.DARKROOM.WEB-DL.mkv`,
	})
	if service != "DARKROOM" {
		t.Fatalf("expected DARKROOM service, got %q", service)
	}
	if longName != "DARKROOM" {
		t.Fatalf("expected DARKROOM long name, got %q", longName)
	}
	if filename == "" {
		t.Fatalf("expected filename to be preserved")
	}
}

type stubRepo struct {
	saved    db.FileMetadata
	existing db.FileMetadata
}

type stubMediaInfo struct{}

func (stubMediaInfo) Export(context.Context, mediainfo.Request) (mediainfo.Result, error) {
	return mediainfo.Result{}, nil
}

type stubSceneDetector struct{}

func (stubSceneDetector) Detect(context.Context, api.PreparedMetadata) (SceneResult, error) {
	return SceneResult{}, nil
}

func (s *stubRepo) GetByPath(context.Context, string) (db.FileMetadata, error) {
	if s.existing.Path != "" {
		return s.existing, nil
	}
	return db.FileMetadata{}, internalerrors.ErrNotFound
}

func (s *stubRepo) Save(_ context.Context, metadata db.FileMetadata) error {
	metadata.UpdatedAt = time.Now().UTC()
	s.saved = metadata
	return nil
}

func (s *stubRepo) GetExternalIDs(context.Context, string) (db.ExternalIDs, error) {
	return db.ExternalIDs{}, internalerrors.ErrNotFound
}

func (s *stubRepo) SaveExternalIDs(context.Context, db.ExternalIDs) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) GetExternalMetadata(context.Context, string) (db.ExternalMetadata, error) {
	return db.ExternalMetadata{}, internalerrors.ErrNotFound
}

func (s *stubRepo) SaveExternalMetadata(context.Context, db.ExternalMetadata) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) GetDVDMediaInfo(context.Context, string) (db.DVDMediaInfo, error) {
	return db.DVDMediaInfo{}, internalerrors.ErrNotFound
}

func (s *stubRepo) SaveDVDMediaInfo(context.Context, db.DVDMediaInfo) error {
	return nil
}

func (s *stubRepo) GetReleaseNameOverrides(context.Context, string) (db.ReleaseNameOverrides, error) {
	return db.ReleaseNameOverrides{}, internalerrors.ErrNotFound
}

func (s *stubRepo) SaveReleaseNameOverrides(context.Context, string, db.ReleaseNameOverrides) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) DeleteReleaseNameOverrides(context.Context, string) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) GetDescriptionOverride(context.Context, string) (db.DescriptionOverride, error) {
	return db.DescriptionOverride{}, internalerrors.ErrNotFound
}

func (s *stubRepo) SaveDescriptionOverride(context.Context, db.DescriptionOverride) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) DeleteDescriptionOverride(context.Context, string) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) ListHistoryEntries(context.Context) ([]db.HistoryEntry, error) {
	return nil, internalerrors.ErrNotImplemented
}

func (s *stubRepo) ListUploadHistoryByPath(context.Context, string) ([]db.UploadRecord, error) {
	return nil, internalerrors.ErrNotImplemented
}

func (s *stubRepo) ListPendingUploads(context.Context) ([]db.UploadRecord, error) {
	return nil, internalerrors.ErrNotImplemented
}

func (s *stubRepo) CreateUploadRecord(context.Context, db.UploadRecord) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) UpdateLatestUploadRecordStatus(context.Context, string, string, string) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) SaveTrackerRuleFailures(context.Context, string, string, []db.TrackerRuleFailure) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) ListTrackerRuleFailuresByPath(context.Context, string) ([]db.TrackerRuleFailure, error) {
	return nil, internalerrors.ErrNotImplemented
}

func (s *stubRepo) GetTrackerTimestamp(context.Context, string) (time.Time, error) {
	return time.Time{}, internalerrors.ErrNotImplemented
}

func (s *stubRepo) SaveTrackerTimestamp(context.Context, db.TrackerTimestamp) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) SaveTrackerMetadata(context.Context, db.TrackerMetadata) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) ListTrackerMetadataByPath(context.Context, string) ([]db.TrackerMetadata, error) {
	return nil, internalerrors.ErrNotImplemented
}

func (s *stubRepo) SaveScreenshot(context.Context, db.Screenshot) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) ListScreenshotsByPath(context.Context, string) ([]db.Screenshot, error) {
	return nil, internalerrors.ErrNotImplemented
}

func (s *stubRepo) DeleteScreenshot(context.Context, string) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) SaveFinalSelections(context.Context, string, []db.ScreenshotFinalSelection) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) ListFinalSelections(context.Context, string) ([]db.ScreenshotFinalSelection, error) {
	return nil, internalerrors.ErrNotImplemented
}

func (s *stubRepo) DeleteFinalSelection(context.Context, string) error {
	return internalerrors.ErrNotImplemented
}
func (s *stubRepo) ReplaceScreenshotSlots(context.Context, string, []db.ScreenshotSlot) error {
	return internalerrors.ErrNotImplemented
}
func (s *stubRepo) ListScreenshotSlotsByPath(context.Context, string) ([]db.ScreenshotSlot, error) {
	return nil, internalerrors.ErrNotImplemented
}
func (s *stubRepo) UpsertScreenshotSlotVariants(context.Context, string, []db.ScreenshotSlotVariant) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) SaveUploadedImages(context.Context, string, string, []db.UploadedImageLink) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) ListUploadedImagesByPath(context.Context, string) ([]db.UploadedImageLink, error) {
	return nil, internalerrors.ErrNotImplemented
}

func (s *stubRepo) DeleteUploadedImage(context.Context, string, string, string) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) GetPlaylistSelection(context.Context, string) (db.PlaylistSelection, error) {
	return db.PlaylistSelection{}, internalerrors.ErrNotImplemented
}

func (s *stubRepo) SavePlaylistSelection(context.Context, string, []string, bool) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) DeletePlaylistSelection(context.Context, string) error {
	return internalerrors.ErrNotImplemented
}

func (s *stubRepo) ListStoredReleasePaths(context.Context) ([]string, error) {
	return nil, internalerrors.ErrNotImplemented
}

func (s *stubRepo) PurgeContentData(context.Context, string) error {
	return internalerrors.ErrNotImplemented
}
