//go:build windows

// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package webserver

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type powershellNativePicker struct{}

func newNativePicker() nativePicker {
	return powershellNativePicker{}
}

func (powershellNativePicker) BrowseFile() (string, error) {
	return runPickerScript(`
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Title = 'Select a file'
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
  [Console]::Out.Write($dialog.FileName)
}
`)
}

func (powershellNativePicker) BrowseFolder() (string, error) {
	return runPickerScript(`
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = 'Select a folder'
$dialog.ShowNewFolderButton = $false
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
  [Console]::Out.Write($dialog.SelectedPath)
}
`)
}

func runPickerScript(script string) (string, error) {
	cmd := exec.Command(
		"powershell",
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-STA",
		"-Command",
		script,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("native browse failed: %s", message)
	}
	return strings.TrimSpace(stdout.String()), nil
}
