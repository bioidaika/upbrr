// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package screenshots

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBundledFFmpegPathPrefersWorkingDirectory(t *testing.T) {
	folder := osFolder()
	if folder == "" {
		t.Skip("unsupported platform")
	}

	root := t.TempDir()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	path := filepath.Join(root, "bin", "ffmpeg", folder, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o755); err != nil {
		t.Fatalf("write bundled ffmpeg: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	got := bundledFFmpegPath()
	if got != path {
		t.Fatalf("bundledFFmpegPath() = %q, want %q", got, path)
	}
}

func TestBundledFFmpegPathReturnsEmptyWhenMissing(t *testing.T) {
	root := t.TempDir()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	if got := bundledFFmpegPath(); got != "" {
		t.Fatalf("bundledFFmpegPath() = %q, want empty string", got)
	}
}
