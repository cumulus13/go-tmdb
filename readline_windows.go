//go:build windows

package main

// On Windows we never enter raw mode — the shared bufio.Scanner path is used.
// isRawModeSupported returns false so readlineRaw skips straight to the scanner.

func isRawModeSupported() bool { return false }
func setRawMode() error        { return nil }
func restoreMode()             {}
