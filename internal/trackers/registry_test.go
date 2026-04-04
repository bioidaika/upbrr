// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package trackers

import (
	"context"
	"testing"

	"github.com/autobrr/upbrr/pkg/api"
)

type stubDefinition struct {
	name string
}

func (s stubDefinition) Name() string {
	return s.name
}

func (s stubDefinition) Upload(context.Context, UploadRequest) (api.UploadSummary, error) {
	return api.UploadSummary{Uploaded: 1}, nil
}

func TestRegistryRegisterLookup(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubDefinition{name: "Blu"}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if _, ok := registry.Lookup("BLU"); !ok {
		t.Fatalf("expected lookup to succeed")
	}
	if _, ok := registry.Lookup("blu"); !ok {
		t.Fatalf("expected lookup to be case-insensitive")
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubDefinition{name: "BLU"}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if err := registry.Register(stubDefinition{name: "blu"}); err == nil {
		t.Fatalf("expected duplicate register error")
	}
}
