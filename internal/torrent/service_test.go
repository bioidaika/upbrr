// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package torrent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	mkbrr "github.com/autobrr/mkbrr/torrent"

	internalerrors "github.com/autobrr/upbrr/internal/errors"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestCreateReusesTorrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	contentPath := filepath.Join(dir, "sample.bin")
	torrentPath := filepath.Join(dir, "sample.torrent")
	createTestTorrent(t, contentPath, torrentPath)

	service := NewService(api.NopLogger{}, t.TempDir())
	result, err := service.Create(context.Background(), api.PreparedMetadata{SourcePath: torrentPath})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Path != torrentPath {
		t.Fatalf("unexpected torrent path: %s", result.Path)
	}
	if result.InfoHash == "" {
		t.Fatalf("expected info hash to be populated")
	}
}

func TestCreateFallbacksToSibling(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "video.mkv")
	sibling := source + ".torrent"
	createTestTorrent(t, source, sibling)

	service := NewService(api.NopLogger{}, t.TempDir())
	result, err := service.Create(context.Background(), api.PreparedMetadata{SourcePath: source})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Path != sibling {
		t.Fatalf("unexpected torrent path: %s", result.Path)
	}
	if result.InfoHash == "" {
		t.Fatalf("expected info hash to be populated")
	}
}

func TestCreateMissingTorrent(t *testing.T) {
	t.Parallel()

	service := NewService(api.NopLogger{}, t.TempDir())
	_, err := service.Create(context.Background(), api.PreparedMetadata{SourcePath: "/missing/file.mkv"})
	if !errors.Is(err, internalerrors.ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestCreateNewTorrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "video.mkv")
	if err := os.WriteFile(source, []byte("data"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tmpRoot := t.TempDir()
	service := NewService(api.NopLogger{}, tmpRoot)
	result, err := service.Create(context.Background(), api.PreparedMetadata{SourcePath: source})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Path == "" {
		t.Fatalf("expected torrent path, got empty")
	}
	if !strings.HasPrefix(result.Path, tmpRoot) {
		t.Fatalf("expected torrent path under tmp root, got %s", result.Path)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("expected torrent file to exist, got %v", err)
	}
	if result.InfoHash == "" {
		t.Fatalf("expected info hash to be populated")
	}
}

func TestCreateHonorsMaxPieceSizeOverride(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "video.mkv")
	content := make([]byte, 10<<20)
	if err := os.WriteFile(source, content, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tmpRoot := t.TempDir()
	service := NewService(api.NopLogger{}, tmpRoot)
	maxPiece := 1
	result, err := service.Create(context.Background(), api.PreparedMetadata{
		SourcePath: source,
		TorrentOverrides: api.TorrentOverrides{
			MaxPieceSizeMiB: &maxPiece,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	torrentMeta, err := metainfo.LoadFromFile(result.Path)
	if err != nil {
		t.Fatalf("load torrent: %v", err)
	}
	info, err := torrentMeta.UnmarshalInfo()
	if err != nil {
		t.Fatalf("unmarshal info: %v", err)
	}
	if info.PieceLength > 1<<20 {
		t.Fatalf("expected piece length <= 1 MiB, got %d", info.PieceLength)
	}
}

func TestCreateNoHashRequiresReusableTorrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "video.mkv")
	if err := os.WriteFile(source, []byte("data"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	service := NewService(api.NopLogger{}, t.TempDir())
	reuseOnly := true
	_, err := service.Create(context.Background(), api.PreparedMetadata{
		SourcePath: source,
		TorrentOverrides: api.TorrentOverrides{
			NoHash: &reuseOnly,
		},
	})
	if !errors.Is(err, internalerrors.ErrNotFound) {
		t.Fatalf("expected nohash to fail without reusable torrent, got %v", err)
	}
}

func TestCreateRehashBypassesReusableTempTorrent(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "video.mkv")
	if err := os.WriteFile(source, []byte("source-data"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	tmpRoot := t.TempDir()
	service := NewService(api.NopLogger{}, tmpRoot)
	meta := api.PreparedMetadata{SourcePath: source}

	tmpTorrentPath, err := TempTorrentPath(tmpRoot, meta, source)
	if err != nil {
		t.Fatalf("temp torrent path: %v", err)
	}
	createTestTorrent(t, source, tmpTorrentPath)

	oldTime := filepath.Join(sourceDir, "marker")
	if err := os.WriteFile(oldTime, []byte("marker"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	past := (mustStat(t, oldTime)).ModTime().Add(-2 * time.Hour)
	if err := os.Chtimes(tmpTorrentPath, past, past); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	rehash := true
	meta.TorrentOverrides = api.TorrentOverrides{Rehash: &rehash}
	result, err := service.Create(context.Background(), meta)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Path != tmpTorrentPath {
		t.Fatalf("expected recreated torrent at temp path %s, got %s", tmpTorrentPath, result.Path)
	}
	if got := mustStat(t, result.Path).ModTime(); !got.After(past) {
		t.Fatalf("expected rehash to recreate torrent, modtime %v was not after %v", got, past)
	}
}

func TestPieceExpForMiB(t *testing.T) {
	t.Parallel()

	cases := map[int]uint{
		1:   20,
		2:   21,
		4:   22,
		8:   23,
		16:  24,
		32:  25,
		64:  26,
		128: 27,
	}

	for input, expected := range cases {
		got, ok := pieceExpForMiB(input)
		if !ok {
			t.Fatalf("expected %d MiB to be supported", input)
		}
		if got != expected {
			t.Fatalf("%d MiB: expected exp %d, got %d", input, expected, got)
		}
	}
	if _, ok := pieceExpForMiB(3); ok {
		t.Fatal("expected unsupported value to return false")
	}
}

func TestApplyTorrentOverridePieceOptionsKeepsUserMax(t *testing.T) {
	t.Parallel()

	maxPiece := 16
	requiredExp := uint(26)

	options := applyTorrentOverridePieceOptions(api.PreparedMetadata{
		TorrentOverrides: api.TorrentOverrides{
			MaxPieceSizeMiB: &maxPiece,
		},
	}, mkbrrPieceOptions{
		maxPieceExp: 27,
		pieceExp:    &requiredExp,
	})

	if options.maxPieceExp != 24 {
		t.Fatalf("expected user max exponent 24, got %d", options.maxPieceExp)
	}
	if options.pieceExp == nil || *options.pieceExp != requiredExp {
		t.Fatalf("expected required piece exponent %d to remain set, got %#v", requiredExp, options.pieceExp)
	}
}

func TestApplyTorrentOverridePieceOptionsCapsToTrackerMax(t *testing.T) {
	t.Parallel()

	maxPiece := 128

	options := applyTorrentOverridePieceOptions(api.PreparedMetadata{
		TorrentOverrides: api.TorrentOverrides{
			MaxPieceSizeMiB: &maxPiece,
		},
	}, mkbrrPieceOptions{
		maxPieceExp: 24,
	})

	if options.maxPieceExp != 24 {
		t.Fatalf("expected tracker max exponent 24, got %d", options.maxPieceExp)
	}
}

func TestCreateReusesAssociatedTempTorrent(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "video.mkv")
	if err := os.WriteFile(source, []byte("source-data"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	tmpRoot := t.TempDir()
	service := NewService(api.NopLogger{}, tmpRoot)

	meta := api.PreparedMetadata{SourcePath: source}
	tmpTorrentPath, err := TempTorrentPath(tmpRoot, meta, source)
	if err != nil {
		t.Fatalf("temp torrent path: %v", err)
	}
	createTestTorrent(t, source, tmpTorrentPath)

	result, err := service.Create(context.Background(), meta)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Path != tmpTorrentPath {
		t.Fatalf("expected temp torrent path %s, got %s", tmpTorrentPath, result.Path)
	}
	if result.InfoHash == "" {
		t.Fatalf("expected info hash to be populated")
	}
}

func TestCreatePrefersClientTorrentOverAssociatedTempTorrent(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "video.mkv")
	if err := os.WriteFile(source, []byte("source-data"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	tmpRoot := t.TempDir()
	service := NewService(api.NopLogger{}, tmpRoot)
	meta := api.PreparedMetadata{SourcePath: source}

	tmpTorrentPath, err := TempTorrentPath(tmpRoot, meta, source)
	if err != nil {
		t.Fatalf("temp torrent path: %v", err)
	}
	createTestTorrent(t, source, tmpTorrentPath)

	clientSource := filepath.Join(sourceDir, "client.bin")
	clientTorrentPath := filepath.Join(sourceDir, "client.torrent")
	createTestTorrent(t, clientSource, clientTorrentPath)

	meta.ClientTorrentPath = clientTorrentPath
	result, err := service.Create(context.Background(), meta)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Path != clientTorrentPath {
		t.Fatalf("expected client torrent path %s, got %s", clientTorrentPath, result.Path)
	}
	if result.InfoHash == "" {
		t.Fatalf("expected info hash to be populated")
	}
}

func TestCreateRegeneratesNonCompliantPTPTorrent(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "movie.mkv")
	content := make([]byte, 70<<20)
	if err := os.WriteFile(source, content, 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	clientTorrentPath := filepath.Join(sourceDir, "client.torrent")
	wrongPiece := uint(16)
	_, err := mkbrr.Create(mkbrr.CreateOptions{
		Path:           source,
		OutputPath:     clientTorrentPath,
		IsPrivate:      true,
		PieceLengthExp: &wrongPiece,
	})
	if err != nil {
		t.Fatalf("create client torrent: %v", err)
	}

	service := NewService(api.NopLogger{}, t.TempDir())
	result, err := service.Create(context.Background(), api.PreparedMetadata{
		SourcePath:        source,
		SourceSize:        int64(len(content)),
		Trackers:          []string{"PTP"},
		ClientTorrentPath: clientTorrentPath,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Path == clientTorrentPath {
		t.Fatalf("expected non-compliant client torrent to be regenerated")
	}

	torrentMeta, err := metainfo.LoadFromFile(result.Path)
	if err != nil {
		t.Fatalf("load torrent: %v", err)
	}
	info, err := torrentMeta.UnmarshalInfo()
	if err != nil {
		t.Fatalf("unmarshal info: %v", err)
	}
	if info.PieceLength != 1<<17 {
		t.Fatalf("expected 128 KiB piece size, got %d", info.PieceLength)
	}
}

func TestPTPPiecePolicyBoundaries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		size uint64
		exp  uint
	}{
		{size: 58 << 20, exp: 16},
		{size: 59 << 20, exp: 17},
		{size: 122 << 20, exp: 17},
		{size: 123 << 20, exp: 18},
		{size: 213 << 20, exp: 18},
		{size: 214 << 20, exp: 19},
		{size: 444 << 20, exp: 19},
		{size: 445 << 20, exp: 20},
		{size: 922 << 20, exp: 20},
		{size: 923 << 20, exp: 21},
		{size: 3977 << 20, exp: 21},
		{size: 3978 << 20, exp: 22},
		{size: 6861 << 20, exp: 22},
		{size: 6862 << 20, exp: 23},
		{size: 14234 << 20, exp: 23},
		{size: 14235 << 20, exp: 24},
	}

	for _, tc := range cases {
		meta := api.PreparedMetadata{Trackers: []string{"PTP"}, SourceSize: int64(tc.size)}
		policy := resolveTrackerPolicy(meta)
		got, ok := policy.requiredPieceExp(meta)
		if !ok {
			t.Fatalf("expected piece exponent for size %d", tc.size)
		}
		if got != tc.exp {
			t.Fatalf("size %d: expected exp %d, got %d", tc.size, tc.exp, got)
		}
	}
}

func TestCreateRegeneratesOversizedANTTorrent(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "movie.mkv")
	file, err := os.Create(source)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	const sourceSize = int64(900 << 20)
	if err := file.Truncate(sourceSize); err != nil {
		_ = file.Close()
		t.Fatalf("truncate source: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close source: %v", err)
	}

	clientTorrentPath := filepath.Join(sourceDir, "client.torrent")
	wrongPiece := uint(16)
	_, err = mkbrr.Create(mkbrr.CreateOptions{
		Path:           source,
		OutputPath:     clientTorrentPath,
		IsPrivate:      true,
		MaxPieceLength: &wrongPiece,
		PieceLengthExp: &wrongPiece,
	})
	if err != nil {
		t.Fatalf("create client torrent: %v", err)
	}
	info, err := os.Stat(clientTorrentPath)
	if err != nil {
		t.Fatalf("stat client torrent: %v", err)
	}
	if info.Size() <= antMaxTorrentBytes {
		t.Fatalf("expected oversized ANT torrent fixture, got %d bytes", info.Size())
	}

	service := NewService(api.NopLogger{}, t.TempDir())
	result, err := service.Create(context.Background(), api.PreparedMetadata{
		SourcePath:        source,
		SourceSize:        sourceSize,
		Trackers:          []string{"ANT"},
		ClientTorrentPath: clientTorrentPath,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Path == clientTorrentPath {
		t.Fatalf("expected oversized client torrent to be regenerated")
	}
	regenerated, err := os.Stat(result.Path)
	if err != nil {
		t.Fatalf("stat regenerated torrent: %v", err)
	}
	if regenerated.Size() > antMaxTorrentBytes {
		t.Fatalf("expected regenerated torrent <= %d bytes, got %d", antMaxTorrentBytes, regenerated.Size())
	}
}

func createTestTorrent(t *testing.T, sourcePath, torrentPath string) {
	t.Helper()

	if err := os.WriteFile(sourcePath, []byte("data"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := mkbrr.Create(mkbrr.CreateOptions{
		Path:       sourcePath,
		OutputPath: torrentPath,
		IsPrivate:  true,
	})
	if err != nil {
		t.Fatalf("create torrent: %v", err)
	}
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info
}
