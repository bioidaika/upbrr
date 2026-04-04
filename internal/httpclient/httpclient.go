// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package httpclient

import (
	"net/http"
	"time"
)

const (
	DefaultTimeout = 45 * time.Second
	UploadTimeout  = 60 * time.Second
)

func New(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &http.Client{Timeout: timeout}
}

func CloneWithTimeout(base *http.Client, timeout time.Duration) *http.Client {
	if base == nil {
		return New(timeout)
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	clone := *base
	clone.Timeout = timeout
	return &clone
}
