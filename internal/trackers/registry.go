// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package trackers

import (
	"errors"
	"fmt"
	"strings"
)

type Registry struct {
	defs map[string]Definition
}

func NewRegistry() *Registry {
	return &Registry{defs: make(map[string]Definition)}
}

func (r *Registry) Register(def Definition) error {
	if def == nil {
		return errors.New("trackers: definition is nil")
	}
	name := strings.ToUpper(strings.TrimSpace(def.Name()))
	if name == "" {
		return errors.New("trackers: definition has empty name")
	}
	if _, exists := r.defs[name]; exists {
		return fmt.Errorf("trackers: definition already registered: %s", name)
	}
	r.defs[name] = def
	return nil
}

func (r *Registry) Lookup(tracker string) (Definition, bool) {
	if r == nil {
		return nil, false
	}
	key := strings.ToUpper(strings.TrimSpace(tracker))
	if key == "" {
		return nil, false
	}
	def, ok := r.defs[key]
	return def, ok
}
