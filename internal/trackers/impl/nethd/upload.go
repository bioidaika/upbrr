// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package nethd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/autobrr/upbrr/internal/cookies"
	"github.com/autobrr/upbrr/internal/httpclient"
	"github.com/autobrr/upbrr/internal/metadata/metautil"
	"github.com/autobrr/upbrr/internal/services/bbcode"
	descriptionunit3d "github.com/autobrr/upbrr/internal/services/description/unit3d"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/internal/trackers/impl/commonhttp"
	"github.com/autobrr/upbrr/pkg/api"
)

const (
	trackerName    = "NETHD"
	defaultBaseURL = "https://nethd.org"
	sourceFlag     = "[nethd.org] NetHD.org"
)

var (
	seoSlugTorrentIDPattern = regexp.MustCompile(`(?i)(?:^|/)[^/?#]*-torrent-(\d+)\.html(?:$|[?#])`)
	seoTorrentSlugIDPattern = regexp.MustCompile(`(?i)(?:^|/)torrent-(\d+)(?:-|\.html|$)`)
)

type uploadState struct {
	baseURL       string
	torrentPath   string
	description   string
	releaseName   string
	fields        map[string]string
	blockedReason string
}

func upload(ctx context.Context, req trackers.UploadRequest) (api.UploadSummary, error) {
	state, trackerCookies, err := prepareUploadState(ctx, req)
	if err != nil {
		return api.UploadSummary{}, err
	}
	if state.blockedReason != "" {
		return api.UploadSummary{}, fmt.Errorf("trackers: %s %s", trackerName, state.blockedReason)
	}

	logger := req.Logger
	if logger == nil {
		logger = api.NopLogger{}
	}
	logger.Infof("trackers: starting upload tracker=%s", trackerName)

	body, contentType, err := commonhttp.BuildMultipartPayload(state.fields, []commonhttp.FileField{{
		FieldName:   "file",
		FileName:    trackerName + ".torrent",
		Path:        state.torrentPath,
		ContentType: "application/x-bittorrent",
	}})
	if err != nil {
		return api.UploadSummary{}, fmt.Errorf("trackers: %w", err)
	}

	uploadURL := state.baseURL + "/takeupload.php"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(body))
	if err != nil {
		return api.UploadSummary{}, fmt.Errorf("trackers: %s request build: %w", trackerName, err)
	}
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("User-Agent", "upbrr")
	commonhttp.ApplyCookies(httpReq, trackerCookies)

	client := httpclient.CloneWithTimeout(&http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}, httpclient.UploadTimeout)
	resp, err := client.Do(httpReq)
	if err != nil {
		return api.UploadSummary{}, fmt.Errorf("trackers: %s upload request: %w", trackerName, err)
	}
	defer resp.Body.Close()

	successCandidate := isRedirectStatus(resp.StatusCode)
	// NETHD reports success entirely through the Location header. Keep the body
	// bounded even for redirects so an unexpected response cannot consume
	// unbounded memory.
	_, responsePreview, err := commonhttp.ReadUploadResponseBody(resp, false, commonhttp.DefaultResponsePreviewBytes)
	if err != nil {
		return api.UploadSummary{}, fmt.Errorf("trackers: %s read upload response: %w", trackerName, err)
	}

	torrentID, torrentURL, resultErr := parseUploadResult(state.baseURL, resp.Header.Get("Location"))
	if successCandidate && resultErr == nil && torrentID != "" {
		logger.Infof("trackers: upload succeeded tracker=%s torrent_id=%s", trackerName, torrentID)
		return api.UploadSummary{
			Uploaded: 1,
			UploadedTorrents: []api.UploadedTorrent{{
				Tracker:     trackerName,
				TorrentID:   torrentID,
				TorrentURL:  torrentURL,
				DownloadURL: state.baseURL + "/download.php?id=" + url.QueryEscape(torrentID),
				TorrentPath: state.torrentPath,
			}},
		}, nil
	}

	_, _ = commonhttp.WriteFailureArtifact(req.Meta, req.AppConfig.MainSettings.DBPath, trackerName, "upload_failure", responsePreview, ".html")
	if resultErr != nil && successCandidate {
		return api.UploadSummary{}, fmt.Errorf("trackers: %s invalid upload redirect: %w", trackerName, resultErr)
	}
	if successCandidate && torrentID == "" {
		return api.UploadSummary{}, errors.New("trackers: NETHD upload redirect missing torrent id")
	}
	return api.UploadSummary{}, commonhttp.UploadHTTPError(trackerName, resp.StatusCode, responsePreview)
}

func buildUploadDryRun(ctx context.Context, req trackers.UploadRequest) (api.TrackerDryRunEntry, error) {
	state, _, err := prepareUploadState(ctx, req)
	if err != nil {
		return api.TrackerDryRunEntry{}, err
	}
	status := "ready"
	message := "dry-run payload generated"
	if state.blockedReason != "" {
		status = "blocked"
		message = state.blockedReason
	}
	return api.TrackerDryRunEntry{
		Tracker:          trackerName,
		Status:           status,
		Message:          message,
		ReleaseName:      state.releaseName,
		DescriptionGroup: "nethd",
		Description:      state.description,
		Endpoint:         state.baseURL + "/takeupload.php",
		Payload:          cloneFields(state.fields),
		Files: []api.TrackerDryRunFile{{
			Field:   "file",
			Path:    state.torrentPath,
			Present: strings.TrimSpace(state.torrentPath) != "",
		}},
	}, nil
}

func prepareUploadState(ctx context.Context, req trackers.UploadRequest) (uploadState, []*http.Cookie, error) {
	baseURL, err := resolveBaseURL(req.TrackerConfig.URL)
	if err != nil {
		return uploadState{}, nil, err
	}
	trackerCookies, err := loadCookies(ctx, req.AppConfig.MainSettings.DBPath, baseURL)
	if err != nil {
		return uploadState{}, nil, fmt.Errorf("trackers: %s load cookies: %w", trackerName, err)
	}

	announceURL := strings.TrimSpace(req.TrackerConfig.AnnounceURL)
	blockedReason := ""
	var torrentPath string
	if announceErr := validateAnnounceURL(announceURL); announceErr != nil {
		blockedReason = announceErr.Error()
		torrentPath, err = trackers.ResolveUploadTorrentPath(req.Meta, req.AppConfig.MainSettings.DBPath)
	} else {
		torrentPath, err = resolveUploadTorrentPath(req.Meta, req.AppConfig.MainSettings.DBPath, announceURL)
	}
	if err != nil {
		return uploadState{}, nil, fmt.Errorf("trackers: %s resolve upload torrent: %w", trackerName, err)
	}

	assets, err := trackers.ResolveDescriptionAssetsWithPrepared(ctx, req.Tracker, req.Meta, req.Repo, req.Logger, req.Assets)
	if err != nil {
		trackers.LogDescriptionAssetResolutionFailure(req.Logger, req.Tracker, err)
		assets = trackers.DescriptionAssets{}
	}
	description := buildDescription(req, assets)
	fields := buildPayload(req.Meta, description)
	return uploadState{
		baseURL:       baseURL,
		torrentPath:   torrentPath,
		description:   description,
		releaseName:   fields["name"],
		fields:        fields,
		blockedReason: blockedReason,
	}, trackerCookies, nil
}

func resolveUploadTorrentPath(meta api.PreparedMetadata, dbPath string, announceURL string) (string, error) {
	preparedPath, err := trackers.ResolveTrackerTorrentArtifactPath(meta, dbPath, trackerName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(meta.TorrentPath) == preparedPath {
		if info, statErr := os.Stat(preparedPath); statErr == nil && !info.IsDir() {
			return preparedPath, nil
		}
	}

	basePath, err := trackers.ResolveUploadTorrentPath(meta, dbPath)
	if err != nil {
		return "", err
	}
	if err := trackers.WritePersonalizedTorrent(basePath, preparedPath, announceURL, "", sourceFlag); err != nil {
		return "", err
	}
	return preparedPath, nil
}

func buildPayload(meta api.PreparedMetadata, description string) map[string]string {
	return map[string]string{
		"name":        resolveName(meta),
		"small_descr": buildSmallDescription(meta),
		"poster":      resolvePoster(meta),
		"type":        "401",
		"subcategory": strconv.Itoa(resolveSubcategoryID(meta)),
		"source":      strconv.Itoa(resolveSourceID(meta)),
		"standard":    strconv.Itoa(resolveStandardID(meta)),
		"url":         resolveIMDbURL(meta),
		"descr":       description,
		"team_sel":    "0",
	}
}

func buildDescription(req trackers.UploadRequest, assets trackers.DescriptionAssets) string {
	if assets.Final {
		return strings.TrimSpace(assets.Description)
	}

	parts := make([]string, 0, 12)
	if header := strings.TrimSpace(req.AppConfig.Description.CustomDescriptionHeader); header != "" {
		parts = append(parts, header)
	}
	if req.AppConfig.Description.AddLogo {
		if logo := resolveLogo(req.Meta); logo != "" {
			parts = append(parts, "[center][img]"+logo+"[/img][/center]")
		}
	}
	if overview := strings.TrimSpace(req.Meta.EpisodeOverview); req.AppConfig.Description.EpisodeOverview && isTV(req.Meta) && overview != "" {
		if title := strings.TrimSpace(req.Meta.EpisodeTitle); title != "" {
			parts = append(parts, "[center]"+title+"[/center]")
		}
		if episodeImage := resolveEpisodeImage(req.Meta); episodeImage != "" {
			parts = append(parts, "[center][img]"+episodeImage+"[/img][/center]")
		}
		parts = append(parts, "[center]"+overview+"[/center]")
	}
	if media := trackers.ReadBDinfoOrMediaInfo(req.AppConfig.MainSettings.DBPath, req.Meta); media != "" {
		parts = append(parts, "[quote]"+media+"[/quote]")
	}
	if description := strings.TrimSpace(assets.Description); description != "" {
		parts = append(parts, description)
	}
	if len(assets.MenuImages) > 0 {
		if header := strings.TrimSpace(req.AppConfig.Description.DiscMenuHeader); header != "" {
			parts = append(parts, header)
		}
		if images := screenshotBlock(assets.MenuImages); images != "" {
			parts = append(parts, images)
		}
	}
	if tonemapHeader := strings.TrimSpace(req.AppConfig.Description.TonemappedHeader); tonemapHeader != "" && descriptionunit3d.ShouldIncludeTonemappedHeader(req.Meta, req.AppConfig, assets.Screenshots) {
		parts = append(parts, tonemapHeader)
	}
	if header := strings.TrimSpace(req.AppConfig.Description.ScreenshotHeader); header != "" {
		parts = append(parts, header)
	}
	if images := screenshotBlock(assets.Screenshots); images != "" {
		parts = append(parts, images)
	}
	if signature := strings.TrimSpace(req.AppConfig.Description.CustomSignature); signature != "" {
		parts = append(parts, signature)
	}
	link, text := descriptionunit3d.UppbrrSignatureLink()
	parts = append(parts, fmt.Sprintf("[right][url=%s][size=1]%s[/size][/url][/right]", link, text))

	description := bbcode.FinalizeTrackerDescription(trackerName, strings.TrimSpace(strings.Join(parts, "\n\n")))
	if req.Meta.Options.Debug {
		descriptionunit3d.SaveDescriptionDebug(req.Meta, trackerName, req.AppConfig.MainSettings.DBPath, description, req.Logger)
	}
	return description
}

func resolveBaseURL(configURL string) (string, error) {
	value := strings.TrimSpace(configURL)
	if value == "" {
		value = defaultBaseURL
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("trackers: %s parse base URL: %w", trackerName, err)
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("trackers: %s base URL must use http or https", trackerName)
	}
	if parsed.Hostname() == "" || parsed.User != nil {
		return "", fmt.Errorf("trackers: %s base URL must contain a host without user info", trackerName)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func loadCookies(ctx context.Context, dbPath string, baseURL string) ([]*http.Cookie, error) {
	host := "nethd.org"
	if parsed, err := url.Parse(baseURL); err == nil && parsed.Hostname() != "" {
		host = parsed.Hostname()
	}
	return wrapTrackerResult(cookies.LoadTrackerHTTPCookies(ctx, dbPath, trackerName, host))
}

func parseUploadResult(baseURL string, location string) (string, string, error) {
	resolved, err := resolveSameOriginURL(baseURL, location)
	if err != nil {
		return "", "", err
	}
	parsed, err := url.Parse(resolved)
	if err != nil {
		return "", "", fmt.Errorf("parse detail redirect: %w", err)
	}

	torrentID := ""
	if strings.EqualFold(strings.TrimRight(parsed.Path, "/"), "/details.php") {
		torrentID = strings.TrimSpace(parsed.Query().Get("id"))
	}
	if torrentID == "" {
		for _, pattern := range []*regexp.Regexp{seoSlugTorrentIDPattern, seoTorrentSlugIDPattern} {
			if match := pattern.FindStringSubmatch(parsed.EscapedPath()); len(match) == 2 {
				torrentID = match[1]
				break
			}
		}
	}
	if !isNumericID(torrentID) {
		return "", "", errors.New("redirect did not contain a numeric torrent id")
	}
	if strings.EqualFold(strings.TrimRight(parsed.Path, "/"), "/details.php") {
		parsed.RawQuery = url.Values{"id": {torrentID}}.Encode()
	} else {
		parsed.RawQuery = ""
	}
	parsed.Fragment = ""
	parsed.RawFragment = ""
	return torrentID, parsed.String(), nil
}

func resolveSameOriginURL(baseURL string, rawURL string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}
	reference, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("parse redirect URL: %w", err)
	}
	if strings.TrimSpace(rawURL) == "" {
		return "", errors.New("empty redirect URL")
	}
	resolved := base.ResolveReference(reference)
	if !strings.EqualFold(base.Scheme, resolved.Scheme) || !strings.EqualFold(base.Host, resolved.Host) {
		return "", errors.New("redirect target is outside the tracker origin")
	}
	return resolved.String(), nil
}

func isRedirectStatus(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func validateAnnounceURL(announceURL string) error {
	if strings.TrimSpace(announceURL) == "" {
		return errors.New("announce_url is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(announceURL))
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || (!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return errors.New("announce_url must be a valid http or https URL")
	}
	foundPasskey := false
	for key, values := range parsed.Query() {
		if !strings.EqualFold(strings.TrimSpace(key), "passkey") {
			continue
		}
		foundPasskey = true
		if len(values) == 0 {
			return errors.New("announce_url must contain a passkey query parameter")
		}
		for _, value := range values {
			passkey := strings.TrimSpace(value)
			if passkey == "" {
				return errors.New("announce_url must contain a passkey query parameter")
			}
			if isPlaceholderPasskey(passkey) {
				return errors.New("announce_url contains a placeholder passkey")
			}
		}
	}
	if !foundPasskey {
		return errors.New("announce_url must contain a passkey query parameter")
	}
	return nil
}

func isPlaceholderPasskey(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.Trim(normalized, "<>[]{}()")
	normalized = strings.NewReplacer("-", "_", " ", "_").Replace(normalized)
	return normalized == "PASSKEY" || normalized == "YOUR_PASSKEY"
}

func resolveName(meta api.PreparedMetadata) string {
	name := metautil.FirstNonEmptyTrimmed(meta.ReleaseName, meta.ReleaseNameNoTag, meta.Release.Title, meta.Filename)
	return strings.Join(strings.Fields(name), " ")
}

func buildSmallDescription(meta api.PreparedMetadata) string {
	parts := make([]string, 0, 4)
	if year := resolveYear(meta); year > 0 {
		parts = append(parts, fmt.Sprintf("(%d)", year))
	}
	if hasVietnameseLanguage(meta.AudioLanguages) {
		parts = append(parts, "(TM/LT)")
	}
	if hasVietnameseLanguage(meta.SubtitleLanguages) {
		parts = append(parts, "(VietSub)")
	}
	if tmdbID := resolveTMDBID(meta); tmdbID > 0 {
		kind := "movie"
		if isTV(meta) {
			kind = "tv"
		}
		parts = append(parts, fmt.Sprintf("%s/%d", kind, tmdbID))
	}
	return strings.Join(parts, " ")
}

func resolveSubcategoryID(meta api.PreparedMetadata) int {
	if isTV(meta) {
		if meta.TVPack {
			return 550
		}
		return 511
	}
	combined := strings.ToLower(strings.Join(resolveGenreKeywords(meta), " "))
	for _, candidate := range []struct {
		keywords []string
		id       int
	}{
		{keywords: []string{"animation", "anime"}, id: 425},
		{keywords: []string{"documentary"}, id: 430},
		{keywords: []string{"horror"}, id: 551},
		{keywords: []string{"sci-fi", "science fiction"}, id: 431},
		{keywords: []string{"thriller"}, id: 427},
		{keywords: []string{"action"}, id: 423},
		{keywords: []string{"comedy"}, id: 424},
		{keywords: []string{"crime"}, id: 429},
		{keywords: []string{"drama"}, id: 432},
		{keywords: []string{"fantasy"}, id: 437},
		{keywords: []string{"war"}, id: 537},
		{keywords: []string{"adventure"}, id: 538},
		{keywords: []string{"sport"}, id: 433},
		{keywords: []string{"musical", "music"}, id: 512},
	} {
		for _, keyword := range candidate.keywords {
			if strings.Contains(combined, keyword) {
				return candidate.id
			}
		}
	}
	return 439
}

func resolveSourceID(meta api.PreparedMetadata) int {
	switch strings.ToUpper(strings.TrimSpace(meta.DiscType)) {
	case "BDMV":
		return 411
	case "DVD":
		return 414
	}
	for _, value := range []string{meta.Type, meta.Release.Type} {
		switch normalizeType(value) {
		case "DISC":
			return 411
		case "REMUX":
			return 555
		case "ENCODE", "WEBRIP":
			return 556
		case "WEBDL":
			return 410
		case "HDTV":
			return 413
		case "DVDRIP":
			return 414
		}
	}
	return 530
}

func resolveStandardID(meta api.PreparedMetadata) int {
	switch strings.ToLower(resolveResolution(meta)) {
	case "4320p":
		return 557
	case "2160p":
		return 419
	case "1080p", "1080i":
		return 415
	case "720p":
		return 416
	case "576p", "576i", "480p", "480i":
		return 418
	default:
		return 418
	}
}

func normalizeType(value string) string {
	return strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToUpper(strings.TrimSpace(value)))
}

func resolveResolution(meta api.PreparedMetadata) string {
	if resolution := strings.TrimSpace(meta.Release.Resolution); resolution != "" {
		return resolution
	}
	lower := strings.ToLower(meta.ReleaseName)
	for _, resolution := range []string{"4320p", "2160p", "1080p", "1080i", "720p", "576p", "576i", "480p", "480i"} {
		if strings.Contains(lower, resolution) {
			return resolution
		}
	}
	return ""
}

func resolveGenreKeywords(meta api.PreparedMetadata) []string {
	values := make([]string, 0, 6+len(meta.ArrGenres))
	if meta.Anime {
		values = append(values, "anime")
	}
	if tmdb := meta.ExternalMetadata.TMDB; tmdb != nil {
		values = append(values, tmdb.Genres, tmdb.Keywords)
	}
	if imdb := meta.ExternalMetadata.IMDB; imdb != nil {
		values = append(values, imdb.Genres)
	}
	values = append(values, meta.ArrGenres...)
	values = append(values, meta.Release.Genre)
	return values
}

func resolveYear(meta api.PreparedMetadata) int {
	if tmdb := meta.ExternalMetadata.TMDB; tmdb != nil && tmdb.Year > 0 {
		return tmdb.Year
	}
	if imdb := meta.ExternalMetadata.IMDB; imdb != nil && imdb.Year > 0 {
		return imdb.Year
	}
	for _, year := range []int{meta.Release.Year, meta.ArrYear, meta.EpisodeYear} {
		if year > 0 {
			return year
		}
	}
	return 0
}

func resolveTMDBID(meta api.PreparedMetadata) int {
	if meta.ExternalIDs.TMDBID > 0 {
		return meta.ExternalIDs.TMDBID
	}
	if tmdb := meta.ExternalMetadata.TMDB; tmdb != nil {
		return tmdb.TMDBID
	}
	return 0
}

func resolvePoster(meta api.PreparedMetadata) string {
	if tmdb := meta.ExternalMetadata.TMDB; tmdb != nil {
		if poster := strings.TrimSpace(tmdb.Poster); poster != "" {
			return poster
		}
		if posterPath := strings.TrimSpace(tmdb.TMDBPosterPath); posterPath != "" {
			return "https://image.tmdb.org/t/p/original/" + strings.TrimPrefix(posterPath, "/")
		}
	}
	if imdb := meta.ExternalMetadata.IMDB; imdb != nil {
		return strings.TrimSpace(imdb.Cover)
	}
	return ""
}

func resolveLogo(meta api.PreparedMetadata) string {
	if meta.ExternalMetadata.TMDB == nil {
		return ""
	}
	logo := metautil.FirstNonEmptyTrimmed(meta.ExternalMetadata.TMDB.TMDBLogo, meta.ExternalMetadata.TMDB.Logo)
	if logo == "" {
		return ""
	}
	if parsed, err := url.Parse(logo); err == nil && parsed.IsAbs() {
		return logo
	}
	return "https://image.tmdb.org/t/p/w300/" + strings.TrimPrefix(logo, "/")
}

func resolveEpisodeImage(meta api.PreparedMetadata) string {
	if meta.TVPack {
		if tvmaze := meta.ExternalMetadata.TVmaze; tvmaze != nil {
			if poster := metautil.FirstNonEmptyTrimmed(tvmaze.PosterMedium, tvmaze.Poster); poster != "" {
				return poster
			}
		}
		if tvdb := meta.ExternalMetadata.TVDB; tvdb != nil {
			if poster := strings.TrimSpace(tvdb.Poster); poster != "" {
				return poster
			}
		}
		if tmdb := meta.ExternalMetadata.TMDB; tmdb != nil {
			return strings.TrimSpace(tmdb.Poster)
		}
		return ""
	}
	if tvdb := meta.ExternalMetadata.TVDB; tvdb != nil {
		return strings.TrimSpace(tvdb.EpisodeImage)
	}
	return ""
}

func resolveIMDbURL(meta api.PreparedMetadata) string {
	imdbID := meta.ExternalIDs.IMDBID
	if imdbID <= 0 && meta.ExternalMetadata.IMDB != nil {
		imdbID = meta.ExternalMetadata.IMDB.IMDBID
	}
	if imdbID <= 0 && meta.ExternalMetadata.TMDB != nil {
		imdbID = meta.ExternalMetadata.TMDB.IMDBID
	}
	if imdbID <= 0 {
		return ""
	}
	return fmt.Sprintf("https://www.imdb.com/title/tt%07d/", imdbID)
}

func hasVietnameseLanguage(languages []string) bool {
	for _, language := range languages {
		switch strings.ToLower(strings.TrimSpace(language)) {
		case "vietnamese", "vi", "vi-vn":
			return true
		}
	}
	return false
}

func isTV(meta api.PreparedMetadata) bool {
	for _, category := range []string{meta.ExternalIDs.Category, meta.MediaInfoCategory, meta.Release.Category} {
		if api.NormalizeCategory(category) == api.CategoryTV {
			return true
		}
	}
	return meta.TVPack || meta.HasTVSeasonEpisodeSignal()
}

func screenshotBlock(images []api.ScreenshotImage) string {
	if len(images) == 0 {
		return ""
	}
	parts := make([]string, 0, len(images))
	for _, image := range images {
		imgURL := metautil.FirstNonEmptyTrimmed(image.ImgURL, image.RawURL)
		rawURL := metautil.FirstNonEmptyTrimmed(image.RawURL, image.WebURL, imgURL)
		if imgURL == "" || rawURL == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("[url=%s][img]%s[/img][/url]", rawURL, imgURL))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[center]\n" + strings.Join(parts, " ") + "\n[/center]"
}

func isNumericID(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func cloneFields(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	maps.Copy(result, input)
	return result
}
