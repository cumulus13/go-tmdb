package main

// readline.go — line input with history navigation.
//
// Architecture:
//   • A single package-level *bufio.Scanner wraps os.Stdin so every call
//     shares the same buffer — no bytes are ever consumed and discarded.
//   • On Unix the raw-mode path reads byte-by-byte via os.Stdin.Read (which
//     works because raw mode disables line-buffering in the kernel).
//   • On Windows setRawMode is a no-op, so we always use the scanner path,
//     which gives simple line-at-a-time input with no buffering hazard.
//   • History navigation (↑↓) works on Unix via raw mode.
//     On Windows the user still gets history by re-typing (good enough for now).

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// sharedScanner is the one scanner for the whole process lifetime.
// Allocated lazily so tests that never call readline don't pay for it.
var sharedScanner *bufio.Scanner

func getScanner() *bufio.Scanner {
	if sharedScanner == nil {
		sharedScanner = bufio.NewScanner(os.Stdin)
		sharedScanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	}
	return sharedScanner
}

// readlineRaw is the entry point used by Session.read.
// promptStr is the coloured prompt string; history / histPos are the session's
// history slice and current position.
func readlineRaw(promptStr string, history *[]string, histPos *int) (string, error) {
	fmt.Print(promptStr + " ")

	if !isTTY() {
		// Piped / redirected — use the shared scanner, no echo needed.
		sc := getScanner()
		if sc.Scan() {
			return strings.TrimRight(sc.Text(), "\r\n"), nil
		}
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "exit", nil // EOF
	}

	// Try raw mode (Unix).  On Windows setRawMode is a no-op that returns nil,
	// so we detect that we're in "fake raw" by checking the platform constant.
	if isRawModeSupported() {
		if err := setRawMode(); err == nil {
			defer restoreMode()
			return readRaw(promptStr, history, histPos)
		}
		restoreMode()
	}

	// Fallback: use the shared scanner (Windows, or Unix raw-mode failure).
	sc := getScanner()
	if sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		fmt.Println() // scanner doesn't echo newline on Windows
		return line, nil
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "exit", nil // EOF / Ctrl+Z on Windows
}

// readRaw does byte-by-byte reading with full line editing.
// Only called when the terminal is actually in raw mode (Unix).
func readRaw(promptStr string, history *[]string, histPos *int) (string, error) {
	var buf []rune
	cursor := 0
	tmpHist := ""
	localHistPos := len(*history)

	promptLen := visibleLen(promptStr) + 1

	redraw := func() {
		fmt.Printf("\r\033[%dC\033[K", promptLen)
		fmt.Print(string(buf))
		if cursor < len(buf) {
			fmt.Printf("\033[%dD", len(buf)-cursor)
		}
	}

	for {
		b := make([]byte, 1)
		n, err := os.Stdin.Read(b)
		if err != nil || n == 0 {
			return string(buf), err
		}
		ch := b[0]

		switch {
		case ch == 3: // Ctrl+C
			fmt.Println()
			return "", fmt.Errorf("interrupt")

		case ch == 4: // Ctrl+D
			if len(buf) == 0 {
				fmt.Println()
				return "exit", nil
			}

		case ch == 13 || ch == 10: // Enter
			fmt.Println()
			return string(buf), nil

		case ch == 127 || ch == 8: // Backspace
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
				redraw()
			}

		case ch == 27: // ESC sequence
			seq := make([]byte, 2)
			os.Stdin.Read(seq)
			if seq[0] != '[' { break }
			switch seq[1] {
			case 'A': // ↑
				if localHistPos > 0 {
					if localHistPos == len(*history) { tmpHist = string(buf) }
					localHistPos--
					buf = []rune((*history)[localHistPos])
					cursor = len(buf)
					redraw()
				}
			case 'B': // ↓
				if localHistPos < len(*history) {
					localHistPos++
					if localHistPos == len(*history) {
						buf = []rune(tmpHist)
					} else {
						buf = []rune((*history)[localHistPos])
					}
					cursor = len(buf)
					redraw()
				}
			case 'C': // →
				if cursor < len(buf) { cursor++; fmt.Print("\033[1C") }
			case 'D': // ←
				if cursor > 0 { cursor--; fmt.Print("\033[1D") }
			case 'H': // Home
				if cursor > 0 { fmt.Printf("\033[%dD", cursor); cursor = 0 }
			case 'F': // End
				if cursor < len(buf) { fmt.Printf("\033[%dC", len(buf)-cursor); cursor = len(buf) }
			case '3': // Delete (ESC [ 3 ~)
				extra := make([]byte, 1)
				os.Stdin.Read(extra)
				if extra[0] == '~' && cursor < len(buf) {
					buf = append(buf[:cursor], buf[cursor+1:]...)
					redraw()
				}
			}

		case ch == 1:  // Ctrl+A
			if cursor > 0 { fmt.Printf("\033[%dD", cursor); cursor = 0 }
		case ch == 5:  // Ctrl+E
			if cursor < len(buf) { fmt.Printf("\033[%dC", len(buf)-cursor); cursor = len(buf) }
		case ch == 11: // Ctrl+K
			buf = buf[:cursor]; fmt.Print("\033[K")
		case ch == 21: // Ctrl+U
			if cursor > 0 { buf = buf[cursor:]; cursor = 0; redraw() }
		case ch == 23: // Ctrl+W
			if cursor > 0 {
				end := cursor
				for cursor > 0 && buf[cursor-1] == ' ' { cursor-- }
				for cursor > 0 && buf[cursor-1] != ' ' { cursor-- }
				buf = append(buf[:cursor], buf[end:]...)
				redraw()
			}
		case ch == 12: // Ctrl+L
			fmt.Print("\033[H\033[2J"); redraw()
		case ch == 9: // Tab
			fmt.Print("\a")

		case ch >= 32: // printable
			r := rune(ch)
			if ch >= 0xC0 { // UTF-8 multi-byte
				var fb []byte
				fb = append(fb, ch)
				var extra int
				if ch >= 0xF0 { extra = 3 } else if ch >= 0xE0 { extra = 2 } else { extra = 1 }
				more := make([]byte, extra)
				os.Stdin.Read(more)
				fb = append(fb, more...)
				r = []rune(string(fb))[0]
			}
			buf = append(buf[:cursor], append([]rune{r}, buf[cursor:]...)...)
			cursor++
			redraw()
		}
	}
}

func visibleLen(s string) int {
	inEsc := false
	count := 0
	for _, r := range s {
		if inEsc { if r == 'm' { inEsc = false }; continue }
		if r == '\033' { inEsc = true; continue }
		count++
	}
	return count
}
