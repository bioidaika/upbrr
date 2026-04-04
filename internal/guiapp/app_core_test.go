// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package guiapp

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/autobrr/upbrr/internal/services/db"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestFetchMetadataReportsCoreValidationFailure(t *testing.T) {
	t.Parallel()

	app := &App{coreInitErr: errors.New("invalid config")}

	_, err := app.FetchMetadata("/tmp/example.mkv", "", api.ExternalIDOverrides{}, api.ReleaseNameOverrides{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "core unavailable") {
		t.Fatalf("expected core unavailable error, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid config") {
		t.Fatalf("expected wrapped validation error, got %v", err)
	}
}

func TestListHistoryUsesRepositoryWhenCoreDisabled(t *testing.T) {
	t.Parallel()

	repo := openGUIAppTestRepo(t)
	ctx := context.Background()
	sourcePath := filepath.Join(t.TempDir(), "Example.mkv")
	updatedAt := time.Now().UTC().Add(-time.Hour)
	createdAt := time.Now().UTC()

	if err := repo.Save(ctx, db.FileMetadata{
		Path:       sourcePath,
		Title:      "Example",
		Source:     "BluRay",
		Resolution: "1080p",
		UpdatedAt:  updatedAt,
	}); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	if err := repo.CreateUploadRecord(ctx, db.UploadRecord{
		SourcePath: sourcePath,
		Tracker:    "HDB",
		Status:     "uploaded",
		CreatedAt:  createdAt,
	}); err != nil {
		t.Fatalf("create upload record: %v", err)
	}

	app := &App{
		repo:        repo,
		coreInitErr: errors.New("invalid config"),
	}

	entries, err := app.ListHistory()
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(entries))
	}
	if entries[0].SourcePath != sourcePath {
		t.Fatalf("unexpected source path: %q", entries[0].SourcePath)
	}
	if entries[0].LatestUploadStatus != "Uploaded" {
		t.Fatalf("expected normalized status, got %q", entries[0].LatestUploadStatus)
	}
}

func TestGetHistoryOverviewUsesRepositoryWhenCoreDisabled(t *testing.T) {
	t.Parallel()

	repo := openGUIAppTestRepo(t)
	ctx := context.Background()
	sourcePath := filepath.Join(t.TempDir(), "Example.mkv")
	updatedAt := time.Now().UTC().Add(-time.Hour)
	createdAt := time.Now().UTC()

	if err := repo.Save(ctx, db.FileMetadata{
		Path:       sourcePath,
		Title:      "Example",
		Source:     "WEB",
		Resolution: "2160p",
		UpdatedAt:  updatedAt,
	}); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	if err := repo.CreateUploadRecord(ctx, db.UploadRecord{
		SourcePath: sourcePath,
		Tracker:    "BHD",
		Status:     "failed",
		CreatedAt:  createdAt,
	}); err != nil {
		t.Fatalf("create upload record: %v", err)
	}

	app := &App{
		repo:        repo,
		coreInitErr: errors.New("invalid config"),
	}

	overview, err := app.GetHistoryOverview(sourcePath)
	if err != nil {
		t.Fatalf("get history overview: %v", err)
	}
	if overview.SourcePath != sourcePath {
		t.Fatalf("unexpected source path: %q", overview.SourcePath)
	}
	if overview.ReleaseTitle != "Example" {
		t.Fatalf("unexpected release title: %q", overview.ReleaseTitle)
	}
	if overview.StatusLabel != "Failed" {
		t.Fatalf("expected failed status label, got %q", overview.StatusLabel)
	}
}

func openGUIAppTestRepo(t *testing.T) *db.SQLiteRepository {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "guiapp.db")
	repo, err := db.OpenWithLogger(repoPath, api.NopLogger{})
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})
	if err := repo.Migrate(); err != nil {
		t.Fatalf("migrate repo: %v", err)
	}
	return repo
}
