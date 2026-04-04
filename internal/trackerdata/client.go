// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package trackerdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/services/bbcode"
	"github.com/autobrr/upbrr/pkg/api"
)

const minTokenLength = 25

type Result struct {
	TrackerID   string
	InfoHash    string
	TMDBID      int
	IMDBID      int
	TVDBID      int
	MALID       int
	Category    string
	Description string
	Images      []bbcode.Image
	Validated   []bbcode.Image
	FileName    string
}

func (r Result) HasIDs() bool {
	return r.TMDBID != 0 || r.IMDBID != 0 || r.TVDBID != 0 || r.MALID != 0
}

func (r Result) HasData() bool {
	if r.HasIDs() {
		return true
	}
	if strings.TrimSpace(r.Description) != "" {
		return true
	}
	if len(r.Images) > 0 || len(r.Validated) > 0 {
		return true
	}
	if strings.TrimSpace(r.InfoHash) != "" {
		return true
	}
	if strings.TrimSpace(r.FileName) != "" {
		return true
	}
	if strings.TrimSpace(r.Category) != "" {
		return true
	}
	return false
}

type Client struct {
	cfg    config.Config
	logger api.Logger
	http   *http.Client

	btnURL     string
	bhdBaseURL string
	ptpURL     string
	hdbURL     string
	antURL     string
}

func NewClient(cfg config.Config, logger api.Logger, httpClient *http.Client) *Client {
	if logger == nil {
		logger = api.NopLogger{}
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		cfg:        cfg,
		logger:     logger,
		http:       httpClient,
		btnURL:     "https://api.broadcasthe.net/",
		bhdBaseURL: "https://beyond-hd.me/api/torrents/",
		ptpURL:     "https://passthepopcorn.me/torrents.php",
		hdbURL:     "https://hdbits.org/api/torrents",
		antURL:     "https://anthelion.me/api.php",
	}
}

func (c *Client) Lookup(
	ctx context.Context,
	tracker string,
	trackerID string,
	meta api.PreparedMetadata,
	searchFileName string,
	onlyID bool,
	keepImages bool,
) (Result, error) {
	normalized := strings.ToUpper(strings.TrimSpace(tracker))
	if IsUnit3DTracker(normalized) {
		return c.lookupUnit3D(ctx, normalized, trackerID, searchFileName, onlyID, keepImages)
	}

	switch normalized {
	case "BTN":
		return c.lookupBTN(ctx, strings.TrimSpace(trackerID))
	case "BHD":
		return c.lookupBHD(ctx, trackerID, meta, searchFileName, onlyID, keepImages)
	case "PTP":
		return c.lookupPTP(ctx, trackerID, meta, searchFileName, onlyID, keepImages)
	case "ANT":
		return c.lookupANT(ctx, meta, searchFileName)
	case "HDB":
		return c.lookupHDB(ctx, trackerID, meta, searchFileName, onlyID, keepImages)
	default:
		return Result{}, nil
	}
}

func (c *Client) trackerCfg(name string) (config.TrackerConfig, bool) {
	if c.cfg.Trackers.Trackers == nil {
		return config.TrackerConfig{}, false
	}
	if cfg, ok := c.cfg.Trackers.Trackers[name]; ok {
		return cfg, true
	}
	for key, cfg := range c.cfg.Trackers.Trackers {
		if strings.EqualFold(key, name) {
			return cfg, true
		}
	}
	return config.TrackerConfig{}, false
}

func shouldUseFolderSearch(meta api.PreparedMetadata) bool {
	if strings.TrimSpace(meta.DiscType) != "" {
		return true
	}
	return len(meta.FileList) != 1
}

func (c *Client) getJSON(ctx context.Context, endpoint string, params url.Values, headers map[string]string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("trackerdata: request: %w", err)
	}
	req.URL.RawQuery = params.Encode()
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trackerdata: request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil
	}
	result := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("trackerdata: decode: %w", err)
	}
	return result, nil
}

func mapValue(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return nil
}

func intValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	case string:
		trimmed := strings.TrimSpace(strings.TrimPrefix(v, "tt"))
		if trimmed == "" {
			return 0
		}
		i, _ := strconv.Atoi(trimmed)
		return i
	default:
		return 0
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func trimExt(value string) string {
	trimmed := strings.TrimSpace(value)
	ext := filepath.Ext(trimmed)
	return strings.TrimSuffix(strings.ToLower(trimmed), strings.ToLower(ext))
}

func imdbFromAny(value any) int {
	raw := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(stringValue(value))), "tt")
	return intValue(raw)
}
