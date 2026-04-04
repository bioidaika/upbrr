// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package api

type ReleaseNameRequest struct {
	Category      string
	Type          string
	Title         string
	AltTitle      string
	Year          int
	ManualYear    int
	Resolution    string
	Audio         string
	Service       string
	Season        string
	Episode       string
	Part          string
	Repack        string
	ThreeD        string
	Tag           string
	Source        string
	UHD           string
	HDR           string
	WebDV         bool
	EpisodeTitle  string
	VideoCodec    string
	VideoEncode   string
	DiscType      string
	Region        string
	DVDSize       string
	Edition       string
	SearchYear    string
	DailyDate     string
	ManualDate    bool
	TMDBDateMatch bool
	NoSeason      bool
	NoYear        bool
	NoAKA         bool
}

type ReleaseNameResult struct {
	NameNoTag     string
	Name          string
	CleanName     string
	MissingFields []string
}

type ReleaseNameOverrides struct {
	Category         *string
	Type             *string
	Source           *string
	Resolution       *string
	Tag              *string
	Service          *string
	Edition          *string
	Season           *string
	Episode          *string
	EpisodeTitle     *string
	ManualYear       *int
	ManualDate       *string
	UseSeasonEpisode *bool
	NoSeason         *bool
	NoYear           *bool
	NoAKA            *bool
	NoTag            *bool
	NoEdition        *bool
	NoDub            *bool
	NoDual           *bool
	DualAudio        *bool
	Region           *string
}
