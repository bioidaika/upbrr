// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package webserver

import (
	"testing"
	"time"
)

func TestStopAllDupeJobsWaitsForWorkers(t *testing.T) {
	backend := &Backend{
		dupes: make(map[string]*dupeCheckJob),
	}

	released := make(chan struct{})
	finished := make(chan struct{})
	backend.dupeWG.Add(1)
	backend.dupes["job-1"] = &dupeCheckJob{
		id: "job-1",
		cancel: func() {
			go func() {
				<-released
				backend.dupeWG.Done()
				close(finished)
			}()
		},
	}

	done := make(chan struct{})
	go func() {
		backend.stopAllDupeJobs()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("expected stopAllDupeJobs to wait for active workers")
	case <-time.After(50 * time.Millisecond):
	}

	close(released)

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected stopAllDupeJobs to return after workers finish")
	}

	<-finished
}

func TestStopAllUploadJobsWaitsForWorkers(t *testing.T) {
	backend := &Backend{
		uploads: make(map[string]*trackerUploadJob),
	}

	released := make(chan struct{})
	finished := make(chan struct{})
	backend.uploadWG.Add(1)
	backend.uploads["job-1"] = &trackerUploadJob{
		id: "job-1",
		cancel: func() {
			go func() {
				<-released
				backend.uploadWG.Done()
				close(finished)
			}()
		},
	}

	done := make(chan struct{})
	go func() {
		backend.stopAllUploadJobs()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("expected stopAllUploadJobs to wait for active workers")
	case <-time.After(50 * time.Millisecond):
	}

	close(released)

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected stopAllUploadJobs to return after workers finish")
	}

	<-finished
}

func TestStopAllLogStreamsWaitsForWorkers(t *testing.T) {
	backend := &Backend{
		streams: make(map[string]*backendLogStream),
	}

	stream := &backendLogStream{
		id:   "stream-1",
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	backend.streams[stream.id] = stream
	backend.streamWG.Add(1)

	released := make(chan struct{})
	go func() {
		defer backend.streamWG.Done()
		<-stream.stop
		<-released
		close(stream.done)
	}()

	done := make(chan struct{})
	go func() {
		backend.stopAllLogStreams()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("expected stopAllLogStreams to wait for active workers")
	case <-time.After(50 * time.Millisecond):
	}

	close(released)

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected stopAllLogStreams to return after workers finish")
	}
}

func TestStopSessionLogStreamsStopsMatchingStreams(t *testing.T) {
	backend := &Backend{
		streams: make(map[string]*backendLogStream),
	}

	makeStream := func(id string, sessionID string) *backendLogStream {
		stream := &backendLogStream{
			id:        id,
			sessionID: sessionID,
			stop:      make(chan struct{}),
			done:      make(chan struct{}),
		}
		backend.streamWG.Add(1)
		go func() {
			defer backend.streamWG.Done()
			<-stream.stop
			close(stream.done)
		}()
		return stream
	}

	backend.streams["stream-1"] = makeStream("stream-1", "session-a")
	backend.streams["stream-2"] = makeStream("stream-2", "session-a")
	backend.streams["stream-3"] = makeStream("stream-3", "session-b")

	backend.StopSessionLogStreams("session-a")

	backend.streamMu.Lock()
	_, hasFirst := backend.streams["stream-1"]
	_, hasSecond := backend.streams["stream-2"]
	_, hasThird := backend.streams["stream-3"]
	backend.streamMu.Unlock()

	if hasFirst || hasSecond {
		t.Fatal("expected session log streams to be removed")
	}
	if !hasThird {
		t.Fatal("expected other session log streams to remain")
	}

	_ = backend.StopLogStream("stream-3")
}
