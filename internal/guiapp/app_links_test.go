// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package guiapp

import "testing"

func TestValidateExternalURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "http url", input: "http://example.com/test", wantErr: false},
		{name: "https url", input: "https://example.com/test?q=1", wantErr: false},
		{name: "trimmed url", input: "  https://example.com/path  ", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "relative", input: "/path", wantErr: true},
		{name: "mailto", input: "mailto:user@example.com", wantErr: true},
		{name: "javascript", input: "javascript:alert(1)", wantErr: true},
		{name: "no host", input: "https:///path", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateExternalURL(tc.input)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for input %q", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
		})
	}
}
