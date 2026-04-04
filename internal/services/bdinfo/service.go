// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package bdinfo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	bdrunner "github.com/autobrr/go-bdinfo/pkg/bdinfo"

	"github.com/autobrr/upbrr/pkg/api"
)

type runRequest struct {
	BDMVPath     string
	PlaylistName string
	ReportPath   string
	Reporter     ProgressReporter
}

var runBDInfo = func(ctx context.Context, req runRequest) (bdrunner.Result, error) {
	settings := bdrunner.DefaultSettings(filepath.Dir(req.ReportPath))
	settings.GenerateStreamDiagnostics = false
	settings.ExtendedStreamDiagnostics = true
	settings.SummaryOnly = true
	settings.GenerateTextSummary = true
	settings.PlaylistOnly = req.PlaylistName

	var reporter func(bdrunner.ProgressEvent)
	if req.Reporter != nil {
		reporter = func(event bdrunner.ProgressEvent) {
			emitProgressEvent(req.Reporter, event)
		}
	}
	return bdrunner.Run(ctx, bdrunner.Options{
		Path:       req.BDMVPath,
		ReportPath: req.ReportPath,
		Settings:   settings,
		OnProgress: reporter,
	})
}

func emitProgressEvent(reporter ProgressReporter, event bdrunner.ProgressEvent) {
	if reporter == nil {
		return
	}
	//nolint:exhaustive // We intentionally emit progress only for user-facing stages.
	switch event.Stage {
	case bdrunner.StageStarting, bdrunner.StageScanning:
		if strings.TrimSpace(event.Path) != "" {
			reporter("Scanning: " + event.Path)
		}
	case bdrunner.StageClipInfo, bdrunner.StagePlaylist, bdrunner.StageStream:
		emitDetailedProgressEvent(reporter, event)
	case bdrunner.StageDiscovered:
		reporter(fmt.Sprintf("Found %d playlists, %d clip infos, %d streams", event.Playlists, event.ClipInfos, event.Streams))
	case bdrunner.StageScanComplete:
		reporter("Scan phase complete")
	case bdrunner.StageRenderingReport:
		reporter("Rendering report")
	case bdrunner.StageDone:
		if event.Elapsed > 0 {
			reporter(fmt.Sprintf("Scan complete in %s", event.Elapsed.Round(1e6)))
		} else {
			reporter("Scan complete")
		}
	}
}

func emitDetailedProgressEvent(reporter ProgressReporter, event bdrunner.ProgressEvent) {
	if reporter == nil {
		return
	}

	stage := strings.ToUpper(string(event.Stage))

	if event.Total > 0 {
		if event.TotalBytes > 0 {
			percentage := float64(event.ProcessedBytes) / float64(event.TotalBytes) * 100
			reporter(fmt.Sprintf("%s: %d/%d (%.1f%%)", stage, event.Completed, event.Total, percentage))
			return
		}
		reporter(fmt.Sprintf("%s: %d/%d", stage, event.Completed, event.Total))
		return
	}

	reporter(stage)
}

// Service handles BDInfo execution and parsing for BDMV discs
type Service struct {
	logger api.Logger
}

type progressReporterKey struct{}

// ProgressReporter receives raw BDInfo progress lines.
type ProgressReporter func(line string)

// WithProgressReporter attaches a progress reporter to context.
func WithProgressReporter(ctx context.Context, reporter ProgressReporter) context.Context {
	if reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, progressReporterKey{}, reporter)
}

func progressReporterFromContext(ctx context.Context) ProgressReporter {
	if ctx == nil {
		return nil
	}
	reporter, _ := ctx.Value(progressReporterKey{}).(ProgressReporter)
	return reporter
}

func normalizePlaylistSelector(playlistFile string) string {
	playlistName := strings.TrimSpace(playlistFile)
	playlistName = strings.ReplaceAll(playlistName, "\\", "/")
	if idx := strings.LastIndex(playlistName, "/"); idx >= 0 {
		playlistName = playlistName[idx+1:]
	}
	playlistName = strings.TrimSpace(playlistName)
	if !strings.HasSuffix(strings.ToUpper(playlistName), ".MPLS") {
		playlistName += ".MPLS"
	}
	return strings.ToUpper(playlistName)
}

// New creates a new BDInfo service
func New(logger api.Logger) *Service {
	if logger == nil {
		logger = api.NopLogger{}
	}
	return &Service{logger: logger}
}

// ExecuteForPlaylist runs the embedded Go BDInfo scanner for a specific playlist and returns the output path.
func (s *Service) ExecuteForPlaylist(ctx context.Context, bdmvPath string, playlistFile string, outputDir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	reporter := progressReporterFromContext(ctx)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("bdinfo: create output dir: %w", err)
	}

	s.logger.Debugf("bdinfo: bdmvPath=%s, playlistFile=%s, outputDir=%s", bdmvPath, playlistFile, outputDir)

	playlistName := normalizePlaylistSelector(playlistFile)
	s.logger.Debugf("bdinfo: normalized playlist name: %s", playlistName)

	outputPath := filepath.Join(outputDir, "BD_SUMMARY_00.txt")
	s.logger.Debugf("bdinfo: running in-process for playlist %s", playlistName)
	result, err := runBDInfo(ctx, runRequest{
		BDMVPath:     bdmvPath,
		PlaylistName: playlistName,
		ReportPath:   outputPath,
		Reporter:     reporter,
	})
	if err != nil {
		s.logger.Debugf("bdinfo: in-process execution failed: %v", err)
		return "", fmt.Errorf("bdinfo: execution failed: %w", err)
	}
	if strings.TrimSpace(result.ReportPath) != "" {
		outputPath = result.ReportPath
	}

	reportText := result.Report
	if strings.TrimSpace(reportText) == "" {
		return "", errors.New("bdinfo: empty report content")
	}

	if err := os.WriteFile(outputPath, []byte(reportText), 0o600); err != nil {
		return "", fmt.Errorf("bdinfo: write output: %w", err)
	}

	s.logger.Debugf("bdinfo: successfully completed for playlist %s", playlistFile)

	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("bdinfo: output not found: %w", err)
	}

	s.logger.Debugf("bdinfo: output file found at %s", outputPath)
	return outputPath, nil
}

// ParseOutput parses BDInfo output and returns structured data
func (s *Service) ParseOutput(filePath string) (map[string]interface{}, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("bdinfo: read output: %w", err)
	}

	text := string(content)
	result := make(map[string]interface{})

	// Extract basic info
	if idx := strings.Index(text, "Disc Title:"); idx >= 0 {
		end := strings.Index(text[idx:], "\n")
		if end > 0 {
			result["title"] = strings.TrimSpace(text[idx+11 : idx+end])
		}
	}

	if idx := strings.Index(text, "Disc Label:"); idx >= 0 {
		end := strings.Index(text[idx:], "\n")
		if end > 0 {
			result["label"] = strings.TrimSpace(text[idx+11 : idx+end])
		}
	}

	if idx := strings.Index(text, "Disc Size:"); idx >= 0 {
		end := strings.Index(text[idx:], "\n")
		if end > 0 {
			result["size"] = strings.TrimSpace(text[idx+10 : idx+end])
		}
	}

	if idx := strings.Index(text, "Length:"); idx >= 0 {
		end := strings.Index(text[idx:], "\n")
		if end > 0 {
			result["length"] = strings.TrimSpace(text[idx+7 : idx+end])
		}
	}

	// Extract summary section
	if idx := strings.Index(text, "QUICK SUMMARY:"); idx >= 0 {
		end := strings.Index(text[idx:], "********************")
		if end > 0 {
			result["summary"] = strings.TrimSpace(text[idx+14 : idx+end])
		}
	}

	s.logger.Debugf("bdinfo: parsed output with %d fields", len(result))
	return result, nil
}
