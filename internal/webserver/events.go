// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package webserver

import (
	"encoding/json"
	"sync"
	"time"
)

type serverEvent struct {
	Name string
	Data []byte
}

type eventHub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan serverEvent]struct{}
	logger      interface {
		Debugf(format string, args ...any)
	}
}

func newEventHub() *eventHub {
	return &eventHub{subscribers: make(map[string]map[chan serverEvent]struct{})}
}

func (h *eventHub) SetLogger(logger interface {
	Debugf(format string, args ...any)
}) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.logger = logger
	h.mu.Unlock()
}

func (h *eventHub) Subscribe(sessionID string) (<-chan serverEvent, func()) {
	ch := make(chan serverEvent, 64)

	h.mu.Lock()
	if _, ok := h.subscribers[sessionID]; !ok {
		h.subscribers[sessionID] = make(map[chan serverEvent]struct{})
	}
	h.subscribers[sessionID][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		if subs, ok := h.subscribers[sessionID]; ok {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(h.subscribers, sessionID)
			}
		}
		h.mu.Unlock()
		close(ch)
	}
}

func (h *eventHub) Emit(sessionID string, name string, payload any) {
	if h == nil || sessionID == "" || name == "" {
		return
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.subscribers[sessionID] {
		select {
		case ch <- serverEvent{Name: name, Data: encoded}:
		default:
			if h.logger != nil {
				h.logger.Debugf("web: dropped event %q for session %q because subscriber buffer was full", name, sessionID)
			}
		}
	}
}

func (h *eventHub) EmitKeepAlive(sessionID string) {
	h.Emit(sessionID, "keepalive", map[string]string{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
