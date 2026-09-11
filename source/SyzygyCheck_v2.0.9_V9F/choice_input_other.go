//go:build !windows

package main

import (
	"bufio"
	"strings"
)

// readMenuInput returns escaped=true for a textual "esc" fallback on
// non-Windows platforms and redirected test input. Windows console builds
// additionally recognise the physical Escape key without requiring Enter.
func readMenuInput(r *bufio.Reader) (string, bool) {
	s := readLine(r)
	if strings.EqualFold(strings.TrimSpace(s), "esc") || s == "\x1b" {
		return "", true
	}
	return s, false
}
