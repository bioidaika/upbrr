// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package thexem

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

func TestMapAbsoluteEpisode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/map/single" {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseForm()
		if r.FormValue("id") != "123" || r.FormValue("absolute") != "43" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"scene":{"season":2,"episode":5}}}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), api.NopLogger{})
	client.baseURL = server.URL

	season, episode, err := client.MapAbsoluteEpisode(context.Background(), 123, 43)
	if err != nil {
		t.Fatalf("map absolute: %v", err)
	}
	if season != 2 || episode != 5 {
		t.Fatalf("unexpected mapping season=%d episode=%d", season, episode)
	}
}

func TestGetSeasonNamesAndMatch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/map/names" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"1":["Anime"],"2":["Anime Season 2","Second Season"]}}`))
	}))
	defer server.Close()

	client := NewClient(server.Client(), api.NopLogger{})
	client.baseURL = server.URL

	names, err := client.GetSeasonNames(context.Background(), 456)
	if err != nil {
		t.Fatalf("get season names: %v", err)
	}
	if len(names[2]) == 0 {
		t.Fatalf("expected names for season 2")
	}

	season, err := client.MatchSeasonByName(context.Background(), 456, "Anime Season 2")
	if err != nil {
		t.Fatalf("match season: %v", err)
	}
	if season != 2 {
		t.Fatalf("expected season 2, got %d", season)
	}
}
