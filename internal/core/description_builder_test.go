// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package core

import (
	"context"
	"strings"
	"testing"

	internalerrors "github.com/autobrr/upbrr/internal/errors"
	"github.com/autobrr/upbrr/internal/services/db"
	"github.com/autobrr/upbrr/pkg/api"
)

type stubDescriptionBuilderTrackers struct {
	called  bool
	preview api.PreparationPreview
}

func (s *stubDescriptionBuilderTrackers) Upload(context.Context, api.PreparedMetadata) (api.UploadSummary, error) {
	return api.UploadSummary{Uploaded: 1}, nil
}

func (s *stubDescriptionBuilderTrackers) BuildPreparation(_ context.Context, meta api.PreparedMetadata, trackers []string) (api.PreparationPreview, error) {
	s.called = true
	if strings.TrimSpace(s.preview.SourcePath) == "" {
		s.preview.SourcePath = meta.SourcePath
	}
	if len(s.preview.Descriptions) == 0 {
		s.preview.Descriptions = []api.PreparationDescription{
			{Trackers: trackers, Description: meta.DescriptionTemplate, DescriptionHTML: "<p>ok</p>"},
		}
	}
	return s.preview, nil
}

func (s *stubDescriptionBuilderTrackers) BuildUploadDryRun(context.Context, api.PreparedMetadata, []string) ([]api.TrackerDryRunEntry, error) {
	return []api.TrackerDryRunEntry{}, nil
}

type stubDescriptionRepo struct {
	stubRepo
	override db.DescriptionOverride
	getErr   error
	saved    []db.DescriptionOverride
	deleted  []string
}

func (s *stubDescriptionRepo) GetDescriptionOverride(context.Context, string) (db.DescriptionOverride, error) {
	if s.getErr != nil {
		return db.DescriptionOverride{}, s.getErr
	}
	if strings.TrimSpace(s.override.Description) == "" {
		return db.DescriptionOverride{}, internalerrors.ErrNotFound
	}
	return s.override, nil
}

func (s *stubDescriptionRepo) SaveDescriptionOverride(_ context.Context, override db.DescriptionOverride) error {
	s.saved = append(s.saved, override)
	return nil
}

func (s *stubDescriptionRepo) DeleteDescriptionOverride(_ context.Context, path string) error {
	s.deleted = append(s.deleted, path)
	return nil
}

func TestFetchDescriptionBuilderPreviewUsesOverride(t *testing.T) {
	t.Parallel()

	repo := &stubDescriptionRepo{override: db.DescriptionOverride{SourcePath: "/tmp/source", Description: "override desc"}}
	trackerSvc := &stubPreparationTrackers{}
	core := &Core{
		logger: api.NopLogger{},
		services: api.ServiceSet{
			Filesystem: stubFilesystem{paths: []string{"/tmp/source"}},
			Trackers:   trackerSvc,
		},
		repo:      repo,
		dupeCache: make(map[string]dupeCacheEntry),
	}

	preview, err := core.FetchDescriptionBuilderPreview(context.Background(), api.Request{Paths: []string{"/tmp/source"}, Mode: api.ModeGUI})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if preview.Description != "override desc" {
		t.Fatalf("expected override description, got %q", preview.Description)
	}
	if !preview.HasOverride {
		t.Fatalf("expected override flag to be true")
	}
	if trackerSvc.called {
		t.Fatalf("expected tracker service not called when override exists")
	}
}

func TestFetchDescriptionBuilderPreviewFallsBackToPrepareInGUI(t *testing.T) {
	t.Parallel()

	repo := &stubDescriptionRepo{}
	trackerSvc := &stubPreparationTrackers{}
	metaSvc := &stubMeta{}
	core := &Core{
		logger: api.NopLogger{},
		services: api.ServiceSet{
			Filesystem: stubFilesystem{paths: []string{"/tmp/source"}},
			Trackers:   trackerSvc,
			Metadata:   metaSvc,
		},
		repo:      repo,
		dupeCache: make(map[string]dupeCacheEntry),
	}

	preview, err := core.FetchDescriptionBuilderPreview(context.Background(), api.Request{
		Paths:   []string{"/tmp/source"},
		Mode:    api.ModeGUI,
		Options: api.UploadOptions{Screens: 1},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if metaSvc.calls != 1 {
		t.Fatalf("expected metadata prepare to be called once, got %d", metaSvc.calls)
	}
	if !trackerSvc.called {
		t.Fatalf("expected tracker preparation to be called")
	}
	if preview.SourcePath != "/tmp/source" {
		t.Fatalf("expected source path to be set, got %q", preview.SourcePath)
	}
}

func TestSaveDescriptionOverrideDeletesOnEmpty(t *testing.T) {
	t.Parallel()

	repo := &stubDescriptionRepo{}
	core := &Core{
		logger: api.NopLogger{},
		services: api.ServiceSet{
			Filesystem: stubFilesystem{paths: []string{"/tmp/source"}},
		},
		repo:      repo,
		dupeCache: make(map[string]dupeCacheEntry),
	}

	err := core.SaveDescriptionOverride(context.Background(), api.Request{Paths: []string{"/tmp/source"}, Mode: api.ModeGUI}, "  ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.deleted) != 1 {
		t.Fatalf("expected delete to be called, got %d", len(repo.deleted))
	}
}

func TestRenderDescriptionReturnsHTML(t *testing.T) {
	t.Parallel()

	core := &Core{logger: api.NopLogger{}}
	value, err := core.RenderDescription(context.Background(), "[b]Example[/b]")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(value, "Example") {
		t.Fatalf("expected rendered HTML to contain content, got %q", value)
	}
}

func TestFetchDescriptionBuilderPreviewSkipsEmptyPreparationPlaceholder(t *testing.T) {
	t.Parallel()

	repo := &stubDescriptionRepo{}
	trackerSvc := &stubDescriptionBuilderTrackers{
		preview: api.PreparationPreview{
			SourcePath: "/tmp/source",
			Descriptions: []api.PreparationDescription{
				{Trackers: []string{"BTN"}, Description: "", DescriptionHTML: ""},
				{Trackers: []string{"BLU"}, Description: "final description", DescriptionHTML: "<p>final description</p>"},
			},
		},
	}
	metaSvc := &stubMeta{}
	core := &Core{
		logger: api.NopLogger{},
		services: api.ServiceSet{
			Filesystem: stubFilesystem{paths: []string{"/tmp/source"}},
			Trackers:   trackerSvc,
			Metadata:   metaSvc,
		},
		repo:      repo,
		dupeCache: make(map[string]dupeCacheEntry),
	}

	preview, err := core.FetchDescriptionBuilderPreview(context.Background(), api.Request{
		Paths:   []string{"/tmp/source"},
		Mode:    api.ModeGUI,
		Options: api.UploadOptions{Screens: 1},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if preview.Description != "final description" {
		t.Fatalf("expected non-empty description to be selected, got %q", preview.Description)
	}
	if preview.DescriptionHTML != "<p>final description</p>" {
		t.Fatalf("expected non-empty rendered description, got %q", preview.DescriptionHTML)
	}
}
