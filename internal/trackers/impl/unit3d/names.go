// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package unit3d

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/languageutil"
	"github.com/autobrr/upbrr/pkg/api"
)

var noGroupTagPattern = regexp.MustCompile(`(?i)-(nogrp|nogroup|unknown|-unk-)`)
var vmfDubPattern = regexp.MustCompile(`(?i)(lồng tiếng|long tieng|\b(?:us|vn)lt\b)`)
var vmfViePattern = regexp.MustCompile(`(?i)(thuyết minh|thuyet minh|\btm\b)`)
var (
	languageTagLookupOnce sync.Once
	languageTagLookup     map[string]language.Tag
)

func buildUnit3DName(tracker string, meta api.PreparedMetadata, cfg config.TrackerConfig) string {
	trackerName := strings.ToUpper(strings.TrimSpace(tracker))
	if trackerName == "RHD" {
		return buildRHDName(meta)
	}

	name := baseReleaseName(meta)
	if name == "" {
		return ""
	}

	switch trackerName {
	case "AITHER":
		return BuildAitherName(meta)
	case "ACM":
		return buildACMName(meta)
	case "CBR":
		return BuildCBRName(meta, cfg.TagForCustomRelease)
	case "DP":
		return buildDPName(name, meta)
	case "LCD":
		return BuildCBRName(meta, cfg.TagForCustomRelease)
	case "LDU":
		return buildLDUName(name, meta)
	case "RF":
		return addNoGroupSuffix(name, meta, "NoGroup")
	case "SAM":
		return BuildCBRName(meta, cfg.TagForCustomRelease)
	case "OE":
		return addNoGroupSuffix(name, meta, "NOGRP")
	case "ULCX":
		return buildULCXName(name, meta)
	case "VMF":
		return buildVMFName(name, meta)
	case "ZNTH":
		return buildZNTHName(name, meta)
	default:
		return name
	}
}

func baseReleaseName(meta api.PreparedMetadata) string {
	name := strings.TrimSpace(meta.ReleaseName)
	if name == "" {
		name = strings.TrimSpace(meta.ReleaseNameNoTag)
	}
	return strings.TrimSpace(strings.Join(strings.Fields(name), " "))
}

func buildDPName(name string, meta api.PreparedMetadata) string {
	audioLabel := resolveDPAudioLabel(meta.AudioLanguages)
	if audioLabel != "" {
		name = strings.Replace(name, "Dual-Audio", audioLabel, 1)
	}
	return strings.TrimSpace(strings.Join(strings.Fields(name), " "))
}

func buildULCXName(name string, meta api.PreparedMetadata) string {
	if strings.EqualFold(strings.TrimSpace(meta.Type), "WEBDL") && (strings.Contains(strings.ToLower(strings.TrimSpace(meta.Edition)), "hybrid") || meta.WebDV) {
		name = strings.Replace(name, "Hybrid ", "", 1)
	}
	return strings.TrimSpace(strings.Join(strings.Fields(name), " "))
}

func buildVMFName(name string, meta api.PreparedMetadata) string {
	hasDub := false
	hasVie := false

	for _, title := range meta.AudioTitles {
		if vmfDubPattern.MatchString(title) {
			hasDub = true
		}
		if vmfViePattern.MatchString(title) {
			hasVie = true
		}
	}

	for _, lang := range meta.AudioLanguages {
		if strings.ToLower(strings.TrimSpace(lang)) == "vietnamese" {
			hasVie = true
		}
	}

	tag := ""
	if hasDub {
		tag = "ViE DUB"
	} else if hasVie {
		tag = "ViE"
	}

	name, existingTag := stripExistingVMFTags(name)
	if existingTag == "ViE DUB" || tag == "ViE DUB" {
		tag = "ViE DUB"
	} else if existingTag == "ViE" {
		tag = "ViE"
	}
	if tag == "" {
		return strings.TrimSpace(strings.Join(strings.Fields(name), " "))
	}

	if anchor, ok := findVMFResolutionAnchor(name, meta); ok {
		return insertVMFTagBeforeToken(name, anchor, tag)
	}
	if anchor, ok := findVMFSourceAnchor(name, meta); ok {
		return insertVMFTagBeforeToken(name, anchor, tag)
	}
	if anchor, ok := findVMFReleaseGroupAnchor(name, meta); ok {
		return insertVMFTagBeforeGroup(name, anchor, tag)
	}

	separator := preferredVMFSeparator(name, len(name))
	return strings.TrimSpace(name + separator + renderVMFTag(tag, separator))
}

type vmfNameToken struct {
	start int
	end   int
	value string
}

// stripExistingVMFTags removes recognized VMF tag tokens before the name is
// rebuilt. This lets old trailing tags and duplicate/conflicting tags collapse
// to one canonical tag without treating title words such as "Movie" or "Vie"
// as tags. ViE DUB is the strongest existing tag and is never downgraded.
func stripExistingVMFTags(name string) (string, string) {
	tokens := tokenizeVMFName(name)
	removed := make([]bool, len(tokens))
	existingTag := ""
	for i, token := range tokens {
		if !isVMFViETagToken(token.value) {
			continue
		}
		removed[i] = true
		if i+1 < len(tokens) && strings.EqualFold(tokens[i+1].value, "DUB") {
			removed[i+1] = true
			existingTag = "ViE DUB"
			continue
		}
		if existingTag == "" {
			existingTag = "ViE"
		}
	}

	if existingTag == "" {
		return name, ""
	}

	kept := make([]int, 0, len(tokens))
	for i := range tokens {
		if !removed[i] {
			kept = append(kept, i)
		}
	}
	if len(kept) == 0 {
		return "", existingTag
	}

	var rebuilt strings.Builder
	first := kept[0]
	if first == 0 {
		rebuilt.WriteString(name[:tokens[first].start])
	}
	rebuilt.WriteString(name[tokens[first].start:tokens[first].end])
	previous := first
	for _, current := range kept[1:] {
		if current == previous+1 {
			rebuilt.WriteString(name[tokens[previous].end:tokens[current].start])
		} else {
			rebuilt.WriteString(bridgeVMFTagGap(name, tokens, current))
		}
		rebuilt.WriteString(name[tokens[current].start:tokens[current].end])
		previous = current
	}
	if previous == len(tokens)-1 {
		rebuilt.WriteString(name[tokens[previous].end:])
	}

	return strings.TrimSpace(rebuilt.String()), existingTag
}

func bridgeVMFTagGap(name string, tokens []vmfNameToken, next int) string {
	if next <= 0 || next >= len(tokens) {
		return ""
	}
	gapBeforeNext := name[tokens[next-1].end:tokens[next].start]
	if strings.Contains(gapBeforeNext, "-") {
		return "-"
	}
	return preferredVMFSeparator(name, tokens[next].start)
}

func isVMFViETagToken(value string) bool {
	// ViE is deliberately mixed-case in the VMF convention. Matching the exact
	// spelling avoids corrupting an uppercase title word such as "VIE".
	return value == "ViE"
}

func findVMFResolutionAnchor(name string, meta api.PreparedMetadata) (int, bool) {
	tokens := tokenizeVMFName(name)
	seen := map[string]struct{}{}
	for _, resolution := range []string{resolveResolution(meta), detectResolution(name)} {
		resolution = strings.TrimSpace(resolution)
		key := strings.ToLower(resolution)
		if resolution == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		for i := len(tokens) - 1; i >= 0; i-- {
			if strings.EqualFold(tokens[i].value, resolution) {
				return tokens[i].start, true
			}
		}
	}
	for i := len(tokens) - 1; i >= 0; i-- {
		if strings.EqualFold(tokens[i].value, "4K") || strings.EqualFold(tokens[i].value, "8K") {
			return tokens[i].start, true
		}
	}
	return 0, false
}

func findVMFSourceAnchor(name string, meta api.PreparedMetadata) (int, bool) {
	tokens := tokenizeVMFName(name)
	if len(tokens) == 0 {
		return 0, false
	}

	for _, source := range []string{meta.Release.Source, meta.Source} {
		sourceTokens := tokenizeVMFName(source)
		if len(sourceTokens) == 0 {
			continue
		}
		values := make([]string, 0, len(sourceTokens))
		for _, token := range sourceTokens {
			values = append(values, token.value)
		}
		if anchor, ok := findVMFTokenSequence(tokens, values); ok {
			return anchor, true
		}
	}

	for i, token := range tokens {
		value := strings.ToLower(token.value)
		switch value {
		case "bluray", "webdl", "webrip", "hdtv", "uhdtv", "dvd", "remux":
			return token.start, true
		case "web":
			if i+1 < len(tokens) && (strings.EqualFold(tokens[i+1].value, "DL") || strings.EqualFold(tokens[i+1].value, "Rip")) {
				return token.start, true
			}
		case "blu":
			if i+1 < len(tokens) && strings.EqualFold(tokens[i+1].value, "Ray") {
				return token.start, true
			}
		case "uhd":
			if i+1 < len(tokens) && (strings.EqualFold(tokens[i+1].value, "BluRay") || strings.EqualFold(tokens[i+1].value, "Blu")) {
				return token.start, true
			}
		}
	}
	return 0, false
}

func findVMFTokenSequence(tokens []vmfNameToken, values []string) (int, bool) {
	if len(values) == 0 || len(values) > len(tokens) {
		return 0, false
	}
	// Technical source tokens normally occur after the title. Searching from
	// the end avoids treating a title word such as "Web" as the source marker.
	for start := len(tokens) - len(values); start >= 0; start-- {
		matched := true
		for offset, value := range values {
			if !strings.EqualFold(tokens[start+offset].value, value) {
				matched = false
				break
			}
		}
		if matched {
			return tokens[start].start, true
		}
	}
	return 0, false
}

func findVMFReleaseGroupAnchor(name string, meta api.PreparedMetadata) (int, bool) {
	groups := []string{
		strings.TrimSpace(meta.Release.Group),
		strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(meta.Tag), "-")),
	}
	for _, group := range groups {
		if group == "" {
			continue
		}
		if anchor, ok := findVMFReleaseGroupStart(name, group); ok {
			return anchor, true
		}
	}
	return 0, false
}

func findVMFReleaseGroupStart(name string, group string) (int, bool) {
	group = strings.TrimSpace(group)
	if group == "" {
		return 0, false
	}

	trimmedName := strings.TrimRightFunc(name, unicode.IsSpace)
	groupStart, ok := foldSuffixStart(trimmedName, group)
	if !ok {
		return 0, false
	}

	boundary := groupStart
	for boundary > 0 {
		r, size := utf8.DecodeLastRuneInString(trimmedName[:boundary])
		if !unicode.IsSpace(r) {
			break
		}
		boundary -= size
	}
	if boundary > 0 && trimmedName[boundary-1] == '-' {
		return boundary - 1, true
	}
	return 0, false
}

func insertVMFTagBeforeToken(name string, anchor int, tag string) string {
	separator := preferredVMFSeparator(name, anchor)
	rendered := renderVMFTag(tag, separator)
	if anchor <= 0 {
		return rendered + separator + name
	}
	return name[:anchor] + rendered + separator + name[anchor:]
}

func insertVMFTagBeforeGroup(name string, anchor int, tag string) string {
	separator := preferredVMFSeparator(name, anchor)
	rendered := renderVMFTag(tag, separator)
	if anchor <= 0 {
		return rendered + name
	}
	return name[:anchor] + separator + rendered + name[anchor:]
}

func renderVMFTag(tag string, separator string) string {
	if separator == "." {
		return strings.ReplaceAll(tag, " ", ".")
	}
	return tag
}

func preferredVMFSeparator(name string, anchor int) string {
	if anchor > 0 && anchor <= len(name) {
		switch name[anchor-1] {
		case '.':
			return "."
		case ' ', '\t':
			return " "
		}
	}

	tokens := tokenizeVMFName(name)
	dots := 0
	spaces := 0
	for i := 1; i < len(tokens); i++ {
		gap := name[tokens[i-1].end:tokens[i].start]
		if strings.Contains(gap, ".") {
			dots++
		}
		if strings.IndexFunc(gap, unicode.IsSpace) >= 0 {
			spaces++
		}
	}
	if dots > spaces {
		return "."
	}
	return " "
}

func tokenizeVMFName(name string) []vmfNameToken {
	tokens := make([]vmfNameToken, 0, len(strings.Fields(name)))
	start := -1
	for index, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start == -1 {
				start = index
			}
			continue
		}
		if start != -1 {
			tokens = append(tokens, vmfNameToken{start: start, end: index, value: name[start:index]})
			start = -1
		}
	}
	if start != -1 {
		tokens = append(tokens, vmfNameToken{start: start, end: len(name), value: name[start:]})
	}
	return tokens
}

func resolveDPAudioLabel(languages []string) string {
	normalized := make(map[string]struct{}, len(languages))
	for _, language := range languages {
		trimmed := strings.TrimSpace(language)
		if trimmed == "" {
			continue
		}
		normalized[strings.ToUpper(trimmed)] = struct{}{}
	}

	switch len(normalized) {
	case 0:
		return ""
	case 1:
		for value := range normalized {
			return value
		}
		return ""
	case 2:
		return "Dual-Audio"
	default:
		return "MULTi"
	}
}

func addNoGroupSuffix(name string, meta api.PreparedMetadata, suffix string) string {
	tag := strings.TrimSpace(strings.TrimPrefix(meta.Tag, "-"))
	normalizedName := noGroupTagPattern.ReplaceAllString(name, "")
	normalizedName = strings.TrimSpace(strings.Join(strings.Fields(normalizedName), " "))
	if tag != "" && !isNoGroupTag(tag) {
		return normalizedName
	}
	if normalizedName == "" {
		return normalizedName
	}
	if strings.HasSuffix(strings.ToUpper(normalizedName), "-"+strings.ToUpper(suffix)) {
		return normalizedName
	}
	return normalizedName + "-" + suffix
}

func buildLDUName(name string, meta api.PreparedMetadata) string {
	catID := resolveUnit3DLDUCategoryID(meta)
	nonEnglishOriginal := !isEnglishLanguageToken(resolveOriginalLanguage(meta))
	isoAudio, nonEnglishAudio := firstAudioLanguageCode(meta.AudioLanguages)
	isoSubtitle := firstSubtitleLanguageCode(meta.SubtitleLanguages)

	if catID == "18" && isoSubtitle != "" {
		return strings.TrimSpace(strings.Join(strings.Fields(name+" [Subs "+isoSubtitle+"]"), " "))
	}

	if !nonEnglishOriginal && !nonEnglishAudio {
		return strings.TrimSpace(strings.Join(strings.Fields(name), " "))
	}

	languageParts := make([]string, 0, 2)
	if isoAudio != "" {
		languageParts = append(languageParts, "["+isoAudio+"]")
	}
	if isoSubtitle != "" {
		languageParts = append(languageParts, "[Subs "+isoSubtitle+"]")
	}
	if len(languageParts) == 0 {
		return strings.TrimSpace(strings.Join(strings.Fields(name), " "))
	}

	return strings.TrimSpace(strings.Join(strings.Fields(name+" "+strings.Join(languageParts, " ")), " "))
}

func firstAudioLanguageCode(languages []string) (string, bool) {
	for _, value := range languages {
		code, english, ok := languageCode(value)
		if ok {
			return code, !english
		}
	}
	return "", false
}

func firstSubtitleLanguageCode(languages []string) string {
	for _, value := range languages {
		code, _, ok := languageCode(value)
		if ok {
			return code
		}
	}
	return ""
}

func languageCode(value string) (string, bool, bool) {
	normalized := languageutil.NormalizeLanguageDisplay(value)
	if normalized == "" {
		normalized = strings.TrimSpace(value)
	}
	tag, ok := parseLanguageTag(normalized)
	if !ok {
		return "", false, false
	}
	base, _ := tag.Base()
	if base.String() == "und" {
		return "", false, false
	}
	code := base.ISO3()
	if code == "" {
		return "", false, false
	}
	return strings.ToUpper(code), isEnglishLanguageTag(base.String()), true
}

func parseLanguageTag(value string) (language.Tag, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return language.Tag{}, false
	}
	if tag, err := language.Parse(trimmed); err == nil && tag != language.Und {
		return tag, true
	}
	normalized := languageutil.NormalizeLanguageDisplay(trimmed)
	if normalized == "" {
		normalized = trimmed
	}
	languageTagLookupOnce.Do(buildLanguageTagLookup)
	tag, ok := languageTagLookup[strings.ToLower(strings.TrimSpace(normalized))]
	if ok {
		return tag, true
	}
	return language.Tag{}, false
}

func buildLanguageTagLookup() {
	languageTagLookup = make(map[string]language.Tag)
	namer := display.Languages(language.English)
	for _, tag := range display.Supported.Tags() {
		name := strings.ToLower(strings.TrimSpace(namer.Name(tag)))
		if name == "" {
			continue
		}
		if _, exists := languageTagLookup[name]; exists {
			continue
		}
		languageTagLookup[name] = tag
	}
}

func resolveOriginalLanguage(meta api.PreparedMetadata) string {
	switch {
	case meta.ExternalMetadata.TMDB != nil && strings.TrimSpace(meta.ExternalMetadata.TMDB.OriginalLanguage) != "":
		return strings.TrimSpace(meta.ExternalMetadata.TMDB.OriginalLanguage)
	case meta.ExternalMetadata.IMDB != nil && strings.TrimSpace(meta.ExternalMetadata.IMDB.OriginalLanguage) != "":
		return strings.TrimSpace(meta.ExternalMetadata.IMDB.OriginalLanguage)
	default:
		return ""
	}
}

func isEnglishLanguageToken(value string) bool {
	normalized := languageutil.NormalizeLanguageDisplay(value)
	if normalized != "" {
		value = normalized
	}
	return isEnglishLanguageTag(strings.TrimSpace(value))
}

func isEnglishLanguageTag(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "english", "en", "eng", "en-us", "en-gb":
		return true
	default:
		return false
	}
}

func isNoGroupTag(tag string) bool {
	value := strings.ToLower(strings.TrimSpace(tag))
	switch value {
	case "nogrp", "nogroup", "unknown", "-unk-":
		return true
	default:
		return false
	}
}

// buildZNTHName applies ZNTH release-name policy before upload.
// TV names drop episode-title text when it appears before the resolution, while
// non-TV names prefer the IMDb year when it disagrees with the parsed release year.
func buildZNTHName(name string, meta api.PreparedMetadata) string {
	category := resolveUnit3DCategory(meta)
	if category == "TV" && strings.TrimSpace(meta.EpisodeTitle) != "" {
		resolution := resolveResolution(meta)
		if resolution != "" {
			name = replaceZNTHEpisodeTitle(name, meta.EpisodeTitle, resolution)
		}
	}

	if category != "TV" {
		imdbYear := 0
		if meta.ExternalMetadata.IMDB != nil {
			imdbYear = meta.ExternalMetadata.IMDB.Year
		}
		year := meta.Release.Year
		if imdbYear > 0 && year > 0 && imdbYear != year {
			name = replaceZNTHMovieYear(name, meta, year, imdbYear)
		}
	}
	return strings.TrimSpace(strings.Join(strings.Fields(name), " "))
}

// replaceZNTHEpisodeTitle removes the episode-title segment only when its
// normalized text appears immediately before a matching resolution token.
func replaceZNTHEpisodeTitle(name string, episodeTitle string, resolution string) string {
	normalizedTitle := normalizeZNTHAlphaNum(episodeTitle)
	if normalizedTitle == "" {
		return name
	}

	for _, resolutionStart := range findZNTHTokenIndexes(name, resolution) {
		titleStart, ok := findZNTHTitleStartBefore(name[:resolutionStart], normalizedTitle)
		if !ok {
			continue
		}
		return name[:titleStart] + name[resolutionStart:]
	}
	return name
}

// findZNTHTitleStartBefore returns the byte offset of the trailing segment in
// prefix whose alphanumeric-normalized text matches normalizedTitle.
func findZNTHTitleStartBefore(prefix string, normalizedTitle string) (int, bool) {
	candidates := []int{0}
	for i, r := range prefix {
		if !isZNTHAlphaNum(r) {
			candidates = append(candidates, i+len(string(r)))
		}
	}

	for i := len(candidates) - 1; i >= 0; i-- {
		start := candidates[i]
		if normalizeZNTHAlphaNum(prefix[start:]) == normalizedTitle {
			return start, true
		}
	}
	return 0, false
}

// replaceZNTHMovieYear replaces the parsed release-year token before the first
// matching resolution token, or before a trailing metadata release-group suffix
// when no resolution is known.
func replaceZNTHMovieYear(name string, meta api.PreparedMetadata, year int, imdbYear int) string {
	yearToken := strconv.Itoa(year)
	yearIndexes := findZNTHTokenIndexes(name, yearToken)
	if len(yearIndexes) == 0 {
		return name
	}

	searchEnd := len(name)
	if resolution := resolveResolution(meta); resolution != "" {
		resolutionIndexes := findZNTHTokenIndexes(name, resolution)
		if len(resolutionIndexes) > 0 {
			searchEnd = resolutionIndexes[0]
		}
	} else if groupStart, ok := findZNTHReleaseGroupStart(name, meta.Release.Group); ok {
		searchEnd = groupStart
	}

	replaceStart := -1
	for _, yearStart := range yearIndexes {
		if yearStart < searchEnd {
			replaceStart = yearStart
		}
	}
	if replaceStart == -1 {
		return name
	}

	replacement := strconv.Itoa(imdbYear)
	return name[:replaceStart] + replacement + name[replaceStart+len(yearToken):]
}

// findZNTHTokenIndexes returns original-string byte offsets for
// case-insensitive token matches bounded by non-alphanumeric ZNTH separators.
func findZNTHTokenIndexes(value string, token string) []int {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}

	tokenRunes := utf8.RuneCountInString(token)
	indexes := []int{}
	for start := range value {
		end, ok := endAfterZNTHRunes(value, start, tokenRunes)
		if !ok {
			break
		}
		if strings.EqualFold(value[start:end], token) && hasZNTHTokenBoundaries(value, start, end) {
			indexes = append(indexes, start)
		}
	}
	return indexes
}

// findZNTHReleaseGroupStart returns the byte offset of a trailing "-group"
// suffix only when group is a real parsed release group.
func findZNTHReleaseGroupStart(name string, group string) (int, bool) {
	group = strings.TrimSpace(group)
	if group == "" || isNoGroupTag(group) {
		return 0, false
	}

	trimmedName := strings.TrimRightFunc(name, unicode.IsSpace)
	groupStart, ok := foldSuffixStart(trimmedName, group)
	if !ok {
		return 0, false
	}

	boundary := groupStart
	for boundary > 0 {
		r, size := utf8.DecodeLastRuneInString(trimmedName[:boundary])
		if !unicode.IsSpace(r) {
			break
		}
		boundary -= size
	}
	if boundary > 0 && trimmedName[boundary-1] == '-' {
		return boundary - 1, true
	}
	return 0, false
}

// foldSuffixStart returns the byte offset where suffix starts when value ends
// with suffix under Unicode case folding.
func foldSuffixStart(value string, suffix string) (int, bool) {
	start := len(value)
	for range suffix {
		if start == 0 {
			return 0, false
		}
		_, size := utf8.DecodeLastRuneInString(value[:start])
		start -= size
	}
	return start, strings.EqualFold(value[start:], suffix)
}

// endAfterZNTHRunes returns the byte offset after count runes from start.
func endAfterZNTHRunes(value string, start int, count int) (int, bool) {
	end := start
	for range count {
		if end >= len(value) {
			return 0, false
		}
		_, size := utf8.DecodeRuneInString(value[end:])
		end += size
	}
	return end, true
}

// hasZNTHTokenBoundaries reports whether start and end are outside adjacent
// letters or digits in value.
func hasZNTHTokenBoundaries(value string, start int, end int) bool {
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(value[:start])
		if isZNTHAlphaNum(r) {
			return false
		}
	}
	if end < len(value) {
		r, _ := utf8.DecodeRuneInString(value[end:])
		if isZNTHAlphaNum(r) {
			return false
		}
	}
	return true
}

// normalizeZNTHAlphaNum lowercases value and drops every non-alphanumeric rune.
func normalizeZNTHAlphaNum(value string) string {
	var b strings.Builder
	for _, r := range value {
		if isZNTHAlphaNum(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func isZNTHAlphaNum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
