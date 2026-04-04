// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

// Package unit3d provides Unit3D API-based tracker upload implementations.
//
// # Overview
//
// This package implements the tracker.Definition interface for Unit3D-based
// trackers such as Aither, BLU, LST, LUME, and others.
//
// # Architecture
//
// The implementation is split into:
//   - definition.go: Tracker registry and routing to specific implementations
//   - upload.go: Core Unit3D upload logic (multipart form, API calls)
//   - <tracker>_name.go: Tracker-specific name formatting rules
//
// # Adding a New Unit3D Tracker
//
// To add support for a new Unit3D tracker (e.g., "EXAMPLE"):
//
// 1. Add the tracker to the known list in definition.go:
//
//	knownUnit3DTrackers = []string{"AITHER", "BLU", "LST", "LUME", "EXAMPLE"}
//
// 2. Create a name formatter in example_name.go:
//
//	func buildExampleName(meta api.PreparedMetadata) string {
//	    name := meta.ReleaseName
//	    // Apply EXAMPLE-specific name formatting rules
//	    return name
//	}
//
// 3. Add the name formatter to the routing in definition.go Upload():
//
//	case "EXAMPLE":
//	    name = buildExampleName(meta)
//
// 4. Add configuration in config.yaml:
//
//	EXAMPLE:
//	  api_key: "your-api-key"
//	  announce_url: "https://example.tracker/announce/PASSKEY"
//	  anon: false
//	  modq: false
//
// 5. Register in impl/registry.go if not auto-registered:
//
//	registry.Register(unit3d.New())
//
// That's it! The core upload logic in upload.go is tracker-agnostic and will
// handle the API interaction, torrent/NFO attachments, and response parsing.
//
// # Testing
//
// Unit tests live in upload_test.go and cover:
//   - Category/type/resolution mapping
//   - Name formatting per tracker
//   - Form payload construction
//   - Response parsing
//
// Run tests via:
//
//	go test ./internal/trackers/impl/unit3d/...
//
// # Python Reference
//
// The Python implementation lives in:
//   - src/trackers/UNIT3D.py (base class)
//   - src/trackers/AITHER.py (example tracker)
//
// When porting logic, maintain the same behavior but use idiomatic Go patterns.
package unit3d
