// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package webserver

import (
	"testing"
	"time"
)

func TestSessionManagerDeletesExpiredSessionsInBackground(t *testing.T) {
	manager := &sessionManager{
		ttl:          time.Minute,
		cleanupEvery: 10 * time.Millisecond,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
		sessions: map[string]session{
			"expired": {
				ID:        "expired",
				Username:  "tester",
				CSRFToken: "csrf",
				ExpiresAt: time.Now().UTC().Add(-time.Second),
			},
		},
	}
	go manager.cleanupLoop()
	defer manager.Close()

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		manager.mu.Lock()
		_, exists := manager.sessions["expired"]
		manager.mu.Unlock()
		if !exists {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("expected expired session to be removed by background cleanup")
}

func TestSessionManagerCloseStopsCleanupLoop(t *testing.T) {
	manager := &sessionManager{
		ttl:          time.Minute,
		cleanupEvery: time.Hour,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
		sessions:     make(map[string]session),
	}
	go manager.cleanupLoop()

	done := make(chan struct{})
	go func() {
		manager.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected Close to stop the cleanup loop")
	}
}
