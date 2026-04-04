// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"strings"

	"github.com/autobrr/upbrr/pkg/api"
)

func siteTIKProfile() unit3DSiteProfile {
	return unit3DSiteProfile{
		resolveTypeID:     resolveUnit3DTIKTypeID,
		resolveCategoryID: resolveUnit3DTIKCategoryID,
	}
}

func resolveUnit3DTIKTypeID(meta api.PreparedMetadata) string {
	discType := resolveTIKDiscType(meta)
	mapping := map[string]string{
		"CUSTOM":    "1",
		"BD100":     "3",
		"BD66":      "4",
		"BD50":      "5",
		"BD25":      "6",
		"NTSC DVD9": "7",
		"NTSC DVD5": "8",
		"PAL DVD9":  "9",
		"PAL DVD5":  "10",
		"3D":        "11",
	}
	return mapping[discType]
}

func resolveTIKDiscType(meta api.PreparedMetadata) string {
	if meta.TrackerSiteOverrides.TIK.DiscType != nil {
		return strings.ToUpper(strings.TrimSpace(*meta.TrackerSiteOverrides.TIK.DiscType))
	}
	if strings.EqualFold(strings.TrimSpace(meta.Is3D), "3D") {
		return "3D"
	}

	releaseName := strings.ToUpper(strings.TrimSpace(meta.ReleaseName))
	source := strings.ToUpper(strings.TrimSpace(meta.Source))
	if source == "" {
		source = strings.ToUpper(strings.TrimSpace(meta.Release.Source))
	}
	combined := releaseName + " " + source

	for _, marker := range []string{"BD100", "BD66", "BD50", "BD25"} {
		if strings.Contains(combined, marker) {
			return marker
		}
	}

	if strings.EqualFold(strings.TrimSpace(meta.DiscType), "DVD") || strings.Contains(combined, "DVD") {
		if strings.Contains(combined, "PAL") {
			if strings.Contains(combined, "DVD5") {
				return "PAL DVD5"
			}
			return "PAL DVD9"
		}
		if strings.Contains(combined, "NTSC") {
			if strings.Contains(combined, "DVD5") {
				return "NTSC DVD5"
			}
			return "NTSC DVD9"
		}
		if strings.Contains(combined, "DVD5") {
			return "NTSC DVD5"
		}
		return "NTSC DVD9"
	}

	if strings.EqualFold(strings.TrimSpace(meta.DiscType), "BDMV") {
		return "CUSTOM"
	}

	return ""
}

func resolveUnit3DTIKCategoryID(meta api.PreparedMetadata) string {
	category := resolveUnit3DCategory(meta)
	foreign := isTIKForeign(meta)
	opera := isTIKOpera(meta)
	asian := isTIKAsian(meta)

	if strings.EqualFold(category, "MOVIE") {
		if foreign {
			return "3"
		}
		if opera {
			return "5"
		}
		if asian {
			return "6"
		}
		return "1"
	}

	if strings.EqualFold(category, "TV") {
		if foreign {
			return "4"
		}
		if opera {
			return "5"
		}
		return "2"
	}

	return resolveUnit3DCategoryID(meta)
}

func isTIKForeign(meta api.PreparedMetadata) bool {
	if meta.TrackerSiteOverrides.TIK.Foreign != nil {
		return *meta.TrackerSiteOverrides.TIK.Foreign
	}
	if meta.ExternalMetadata.TMDB != nil {
		original := strings.ToLower(strings.TrimSpace(meta.ExternalMetadata.TMDB.OriginalLanguage))
		if original != "" && original != "en" {
			return true
		}
	}
	return !hasEnglishLanguage(meta.AudioLanguages) && !hasEnglishLanguage(meta.SubtitleLanguages)
}

func isTIKOpera(meta api.PreparedMetadata) bool {
	if meta.TrackerSiteOverrides.TIK.Opera != nil {
		return *meta.TrackerSiteOverrides.TIK.Opera
	}
	candidates := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(meta.Release.Genre),
		resolveTMDBGenres(meta),
		resolveIMDBGenres(meta),
		resolveKeywords(meta),
	}, ","))
	return strings.Contains(candidates, "opera") || strings.Contains(candidates, "musical")
}

func isTIKAsian(meta api.PreparedMetadata) bool {
	if meta.TrackerSiteOverrides.TIK.Asian != nil {
		return *meta.TrackerSiteOverrides.TIK.Asian
	}
	if meta.ExternalMetadata.TMDB == nil {
		return false
	}

	asianCountries := map[string]bool{
		"JP": true, "KR": true, "CN": true, "HK": true, "TW": true,
		"TH": true, "VN": true, "IN": true, "ID": true, "MY": true,
		"PH": true, "SG": true,
	}
	for _, country := range meta.ExternalMetadata.TMDB.OriginCountry {
		if asianCountries[strings.ToUpper(strings.TrimSpace(country))] {
			return true
		}
	}

	asianLanguages := map[string]bool{
		"ja": true, "ko": true, "zh": true, "th": true, "vi": true,
		"hi": true, "ta": true, "te": true, "ml": true, "id": true,
		"ms": true,
	}
	original := strings.ToLower(strings.TrimSpace(meta.ExternalMetadata.TMDB.OriginalLanguage))
	return asianLanguages[original]
}
