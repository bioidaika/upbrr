// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package webserver

import (
	"fmt"
	"testing"
	"time"
)

func TestPruneCompletedDupeJobsLockedKeepsNewestCompleted(t *testing.T) {
	backend := &Backend{
		dupes: make(map[string]*dupeCheckJob),
	}

	active := &dupeCheckJob{id: "active", status: "running"}
	backend.dupes[active.id] = active

	now := time.Now().UTC()
	for idx := 0; idx < 3; idx++ {
		id := fmt.Sprintf("dupe-%d", idx)
		backend.dupes[id] = &dupeCheckJob{
			id:         id,
			status:     "completed",
			finishedAt: now.Add(time.Duration(idx) * time.Minute),
		}
	}

	backend.pruneCompletedDupeJobsLocked(2)

	if _, ok := backend.dupes["dupe-0"]; ok {
		t.Fatal("expected oldest completed dupe job to be pruned")
	}
	if _, ok := backend.dupes["dupe-1"]; !ok {
		t.Fatal("expected newer completed dupe job to remain")
	}
	if _, ok := backend.dupes["dupe-2"]; !ok {
		t.Fatal("expected newest completed dupe job to remain")
	}
	if _, ok := backend.dupes[active.id]; !ok {
		t.Fatal("expected active dupe job to remain")
	}
}

func TestPruneCompletedUploadJobsLockedKeepsNewestCompleted(t *testing.T) {
	backend := &Backend{
		uploads: make(map[string]*trackerUploadJob),
	}

	active := &trackerUploadJob{id: "active", status: "running"}
	backend.uploads[active.id] = active

	now := time.Now().UTC()
	for idx := 0; idx < 3; idx++ {
		id := fmt.Sprintf("upload-%d", idx)
		backend.uploads[id] = &trackerUploadJob{
			id:         id,
			status:     "completed",
			finishedAt: now.Add(time.Duration(idx) * time.Minute),
		}
	}

	backend.pruneCompletedUploadJobsLocked(2)

	if _, ok := backend.uploads["upload-0"]; ok {
		t.Fatal("expected oldest completed upload job to be pruned")
	}
	if _, ok := backend.uploads["upload-1"]; !ok {
		t.Fatal("expected newer completed upload job to remain")
	}
	if _, ok := backend.uploads["upload-2"]; !ok {
		t.Fatal("expected newest completed upload job to remain")
	}
	if _, ok := backend.uploads[active.id]; !ok {
		t.Fatal("expected active upload job to remain")
	}
}
