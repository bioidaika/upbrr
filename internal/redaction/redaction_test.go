// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package redaction

import "testing"

func TestRedactValueURLPatterns(t *testing.T) {
	t.Parallel()

	input := "https://tracker.example/0123456789abcdef/announce?passkey=secret&token=abc&info_hash=deadbeef"
	output := RedactValue(input, nil)

	if output == input {
		t.Fatalf("expected redaction, got %q", output)
	}
	if contains(output, "0123456789abcdef") {
		t.Fatalf("expected passkey redacted, got %q", output)
	}
	if contains(output, "secret") || contains(output, "token=abc") {
		t.Fatalf("expected query params redacted, got %q", output)
	}
}

func TestRedactValueProxyPath(t *testing.T) {
	t.Parallel()

	input := "https://example.com/proxy/secret/api/v2/torrents"
	output := RedactValue(input, nil)

	if contains(output, "/proxy/secret/") {
		t.Fatalf("expected proxy secret redacted, got %q", output)
	}
}

func TestRedactPrivateInfoJSON(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"token":   "abc",
		"nested":  map[string]any{"password": "secret"},
		"entries": []any{"passkey", "value"},
	}

	redacted := RedactPrivateInfo(input, nil).(map[string]any)
	if redacted["token"] != "[REDACTED]" {
		t.Fatalf("expected token redacted, got %#v", redacted["token"])
	}
	nested := redacted["nested"].(map[string]any)
	if nested["password"] != "[REDACTED]" {
		t.Fatalf("expected password redacted, got %#v", nested["password"])
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) > 0 && (stringIndex(haystack, needle) >= 0)
}

func stringIndex(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
