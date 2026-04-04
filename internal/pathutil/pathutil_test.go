// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package pathutil

import "testing"

func TestBaseHandlesMixedPlatformPaths(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "windows file", input: `D:\Movies\Movie.Name.2024.mkv`, want: "Movie.Name.2024.mkv"},
		{name: "windows dir", input: `D:\Movies\1982 - Fitzcarraldo [DVD9.PAL]`, want: "1982 - Fitzcarraldo [DVD9.PAL]"},
		{name: "posix file", input: `/media/movies/Movie.Name.2024.mkv`, want: "Movie.Name.2024.mkv"},
		{name: "mixed separators", input: `D:\Movies/subdir\Movie.Name.2024.mkv`, want: "Movie.Name.2024.mkv"},
		{name: "bare name", input: `Movie.Name.2024.mkv`, want: "Movie.Name.2024.mkv"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Base(tc.input); got != tc.want {
				t.Fatalf("Base(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCleanNormalizesSeparators(t *testing.T) {
	if got := Clean(`D:\Movies\subdir\Movie.Name.2024.mkv`); got != "D:/Movies/subdir/Movie.Name.2024.mkv" {
		t.Fatalf("unexpected cleaned path: %q", got)
	}
}
