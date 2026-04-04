// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metadata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestContainerFromMetaDisc(t *testing.T) {
	meta := api.PreparedMetadata{DiscType: "BDMV"}
	if got := containerFromMeta(meta); got != "m2ts" {
		t.Fatalf("expected m2ts, got %q", got)
	}
	meta.DiscType = "DVD"
	if got := containerFromMeta(meta); got != "vob" {
		t.Fatalf("expected vob, got %q", got)
	}
}

func TestContainerFromMetaFileList(t *testing.T) {
	dir := t.TempDir()
	small := filepath.Join(dir, "small.mkv")
	large := filepath.Join(dir, "large.mp4")
	if err := os.WriteFile(small, []byte("small"), 0o600); err != nil {
		t.Fatalf("write small: %v", err)
	}
	if err := os.WriteFile(large, []byte("this is larger"), 0o600); err != nil {
		t.Fatalf("write large: %v", err)
	}
	meta := api.PreparedMetadata{FileList: []string{small, large}}
	if got := containerFromMeta(meta); got != "mp4" {
		t.Fatalf("expected mp4, got %q", got)
	}
}

func TestVideoEncodeFromMedia(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Video", "Format": "AVC", "Format_Profile": "High 10", "Encoded_Library_Settings": "ref=4", "BitDepth": "10"},
	}
	encode, codec, hasSettings, bitDepth := videoEncodeFromMedia(doc, "ENCODE")
	if encode != "Hi10P x264" {
		t.Fatalf("expected Hi10P x264, got %q", encode)
	}
	if codec != "AVC" {
		t.Fatalf("expected AVC, got %q", codec)
	}
	if !hasSettings {
		t.Fatalf("expected encode settings true")
	}
	if bitDepth != "10" {
		t.Fatalf("expected bit depth 10, got %q", bitDepth)
	}
}

func TestVideoEncodeFromMediaMissingBitDepthDefaultsZero(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Video", "Format": "AVC", "Encoded_Library_Settings": "ref=4"},
	}
	_, _, _, bitDepth := videoEncodeFromMedia(doc, "ENCODE")
	if bitDepth != "0" {
		t.Fatalf("expected bit depth 0 when missing, got %q", bitDepth)
	}
}

func TestVideoEncodeFromMediaMPEG4VisualTypeGating(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Video", "Format": "MPEG-4 Visual", "Encoded_Library_Name": "Xvid 1.3.7"},
	}

	encode, _, _, _ := videoEncodeFromMedia(doc, "ENCODE")
	if encode != "XviD" {
		t.Fatalf("expected XviD for ENCODE, got %q", encode)
	}

	encode, _, _, _ = videoEncodeFromMedia(doc, "WEBDL")
	if encode != "" {
		t.Fatalf("expected empty encode for WEBDL MPEG-4 Visual, got %q", encode)
	}
}

func TestVideoEncodeFromMediaHDTVSettingsUseX264(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Video", "Format": "AVC", "Encoded_Library_Settings": "ref=4"},
	}
	encode, _, hasSettings, _ := videoEncodeFromMedia(doc, "HDTV")
	if encode != "x264" {
		t.Fatalf("expected x264 for HDTV with encode settings, got %q", encode)
	}
	if !hasSettings {
		t.Fatalf("expected encode settings true")
	}
}

func TestVideoEncodeFromMediaMPEGVideoCodecVersion(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Video", "Format": "MPEG Video", "Format_Version": "2"},
	}
	_, codec, _, _ := videoEncodeFromMedia(doc, "ENCODE")
	if codec != "MPEG-2" {
		t.Fatalf("expected MPEG-2 codec, got %q", codec)
	}
}

func TestVideoEncodeFromMediaPassthroughCodecs(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Video", "Format": "AV1"},
	}
	encode, codec, _, _ := videoEncodeFromMedia(doc, "ENCODE")
	if encode != "AV1" {
		t.Fatalf("expected AV1 encode passthrough, got %q", encode)
	}
	if codec != "AV1" {
		t.Fatalf("expected AV1 codec passthrough, got %q", codec)
	}
}

func TestHDRFromMedia(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Video", "colour_primaries": "BT.2020", "HDR_Format": "HDR10+"},
	}
	if got := hdrFromMedia(doc, nil); got != "HDR10+" {
		t.Fatalf("expected HDR10+, got %q", got)
	}
}

func TestValidateMediaInfoSettings(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format": "AAC"},
		{"@type": "Video", "Encoded_Library_Settings": "cabac=1"},
	}
	if !validateMediaInfoSettings(doc) {
		t.Fatalf("expected mediainfo settings validation true")
	}
}

func TestAudioFromMedia(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format_Commercial": "Dolby Digital Plus", "Channels": "6", "ChannelLayout": "L R C LFE Ls Rs"},
	}
	audio, channels, commentary := audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if audio != "DD+ 5.1" {
		t.Fatalf("expected DD+ 5.1, got %q", audio)
	}
	if channels != "5.1" {
		t.Fatalf("expected channels 5.1, got %q", channels)
	}
	if commentary {
		t.Fatalf("expected commentary false")
	}
}

func TestAudioFromMediaStreamOrderAndCompatibility(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "StreamOrder": "0", "Format": "DTS", "Format_AdditionalFeatures": "XLL", "Channels": "6", "ChannelLayout": "L R C LFE Ls Rs", "Title": "Compatibility Track"},
		{"@type": "Audio", "StreamOrder": "1", "Format": "E-AC-3", "Channels": "6", "ChannelLayout": "L R C LFE Ls Rs"},
		{"@type": "Audio", "StreamOrder": "2", "Format": "AAC", "Channels": "2", "ChannelLayout": "L R", "Title": "Director Commentary"},
	}
	audio, channels, commentary := audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if audio != "DD+ 5.1" {
		t.Fatalf("expected DD+ 5.1, got %q", audio)
	}
	if channels != "5.1" {
		t.Fatalf("expected channels 5.1, got %q", channels)
	}
	if !commentary {
		t.Fatalf("expected commentary true when any commentary track exists")
	}
}

func TestNormalizeAudioFormatMappings(t *testing.T) {
	tests := []struct {
		name     string
		track    map[string]any
		expected string
	}{
		{name: "ac3", track: map[string]any{"Format": "AC-3"}, expected: "DD"},
		{name: "eac3", track: map[string]any{"Format": "E-AC-3"}, expected: "DD+"},
		{name: "mlp-fba", track: map[string]any{"Format": "MLP FBA"}, expected: "TrueHD"},
		{name: "flac", track: map[string]any{"Format": "FLAC"}, expected: "FLAC"},
		{name: "opus", track: map[string]any{"Format": "Opus"}, expected: "Opus"},
		{name: "vorbis", track: map[string]any{"Format": "Vorbis"}, expected: "VORBIS"},
		{name: "pcm", track: map[string]any{"Format": "PCM"}, expected: "LPCM"},
		{name: "dolby-plus-audio", track: map[string]any{"Format": "Dolby Digital Plus Audio"}, expected: "DD+"},
		{name: "commercial-priority", track: map[string]any{"Format": "DTS", "Format_Commercial": "DTS-HD Master Audio"}, expected: "DTS-HD MA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeAudioFormat(tt.track); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestAudioFromMediaDTSAudioExtraSuffixes(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format": "DTS", "Format_AdditionalFeatures": "XLL", "Channels": "6", "ChannelLayout": "L R C LFE Ls Rs"},
	}
	audio, _, _ := audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if audio != "DTS-HD MA 5.1" {
		t.Fatalf("expected DTS-HD MA 5.1, got %q", audio)
	}

	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format": "DTS", "Format_AdditionalFeatures": "XLL X", "Channels": "6", "ChannelLayout": "L R C LFE Ls Rs"},
	}
	audio, _, _ = audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if audio != "DTS:X 5.1" {
		t.Fatalf("expected DTS:X 5.1, got %q", audio)
	}
}

func TestApplyDTSAudioAdditionalESRequiresExactMatch(t *testing.T) {
	if got := applyDTSAudioAdditional("DTS", "ES"); got != "DTS-ES" {
		t.Fatalf("expected DTS-ES, got %q", got)
	}
	if got := applyDTSAudioAdditional("DTS", "ES Matrix"); got != "DTS" {
		t.Fatalf("expected DTS for non-exact ES additional feature, got %q", got)
	}
}

func TestAudioFromMediaMPEGLayerDetection(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format": "MPEG Audio", "Format_Profile": "Layer 2", "Channels": "2", "ChannelLayout": "L R"},
	}
	audio, _, _ := audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if audio != "MP2 2.0" {
		t.Fatalf("expected MP2 2.0, got %q", audio)
	}

	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format": "MPEG Audio", "Format_Profile": "Layer 3", "Channels": "2", "ChannelLayout": "L R"},
	}
	audio, _, _ = audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if audio != "MP3 2.0" {
		t.Fatalf("expected MP3 2.0, got %q", audio)
	}
}

func TestAudioFromMediaDDSevenPointOneCorrectsToDDPlus(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format": "AC-3", "Channels": "8", "ChannelLayout": "L R C LFE Ls Rs Lb Rb"},
	}
	audio, channels, _ := audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if channels != "7.1" {
		t.Fatalf("expected channels 7.1, got %q", channels)
	}
	if audio != "DD+ 7.1" {
		t.Fatalf("expected DD+ 7.1, got %q", audio)
	}
}

func TestAudioFromMediaImmersiveIndicators(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format": "E-AC-3", "Format_AdditionalFeatures": "16-ch", "Channels": "8", "ChannelLayout": "L R C LFE Ls Rs Lb Rb"},
	}
	audio, _, _ := audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if audio != "DD+ Atmos 7.1" {
		t.Fatalf("expected DD+ Atmos 7.1, got %q", audio)
	}

	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format": "DTS", "Channels": "6", "ChannelLayout": "L R C LFE Ls Rs", "Title": "Auro3D Presentation"},
	}
	audio, _, _ = audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if audio != "DTS Auro3D 5.1" {
		t.Fatalf("expected DTS Auro3D 5.1, got %q", audio)
	}

	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "StreamOrder": "0", "Format": "DTS", "Channels": "6", "ChannelLayout": "L R C LFE Ls Rs", "Title": "Compatibility Track Auro3D"},
		{"@type": "Audio", "StreamOrder": "1", "Format": "E-AC-3", "Channels": "6", "ChannelLayout": "L R C LFE Ls Rs"},
	}
	audio, _, _ = audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if audio != "DD+ Auro3D 5.1" {
		t.Fatalf("expected DD+ Auro3D 5.1, got %q", audio)
	}
}

func TestAudioFromMediaObjectBasedLayoutAssumesLFE(t *testing.T) {
	doc := mediaInfoDoc{}
	doc.Media.Track = []map[string]any{
		{"@type": "Audio", "Format": "TrueHD", "Channels": "8", "ChannelLayout": "L R C Ls Rs Lb Rb Object Based"},
	}

	audio, channels, _ := audioFromMedia(api.PreparedMetadata{}, doc, nil)
	if channels != "7.1" {
		t.Fatalf("expected object-based layout to map to 7.1, got %q", channels)
	}
	if audio != "TrueHD 7.1" {
		t.Fatalf("expected TrueHD 7.1, got %q", audio)
	}
}

func TestResolveService(t *testing.T) {
	meta := api.PreparedMetadata{SourcePath: "/data/Show.Name.2024.NF.WEB-DL.mkv"}
	service, longName, filename := resolveService(meta)
	if service != "NF" {
		t.Fatalf("expected NF, got %q", service)
	}
	if longName != "Netflix" {
		t.Fatalf("expected Netflix, got %q", longName)
	}
	if filename == "" {
		t.Fatalf("expected filename to be populated")
	}
}

func TestApplyMetadataOverrides(t *testing.T) {
	distributor := "Criterion"
	trueValue := true
	falseValue := false

	meta := api.PreparedMetadata{
		Anime:           true,
		StreamOptimized: 0,
		MetadataOverrides: api.MetadataOverrides{
			Distributor:     &distributor,
			PersonalRelease: &trueValue,
			Commentary:      &trueValue,
			WebDV:           &trueValue,
			StreamOptimized: &trueValue,
			Anime:           &falseValue,
		},
	}

	applyMetadataOverrides(&meta)

	if meta.Distributor != "Criterion" {
		t.Fatalf("expected distributor override, got %q", meta.Distributor)
	}
	if !meta.PersonalRelease {
		t.Fatalf("expected personal release override")
	}
	if !meta.HasCommentary {
		t.Fatalf("expected commentary override")
	}
	if !meta.WebDV {
		t.Fatalf("expected webdv override")
	}
	if meta.StreamOptimized != 1 {
		t.Fatalf("expected stream override to set 1, got %d", meta.StreamOptimized)
	}
	if meta.Anime {
		t.Fatalf("expected anime override to set false")
	}
}
