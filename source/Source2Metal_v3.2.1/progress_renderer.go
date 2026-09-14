package main

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

// progressRenderer owns one active console progress line. It deliberately
// keeps every render within the current visible console width. When the user
// resizes the window, the previous line may have been reflowed over multiple
// rows by Windows; before drawing the next state we clear all rows that can
// belong to that previous render and redraw only the current state.
type progressRenderer struct {
	mu       sync.Mutex
	active   bool
	lastText string
}

func newProgressRenderer() *progressRenderer { return &progressRenderer{} }

func (r *progressRenderer) Render(build func(width int) string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	width := visibleConsoleWidth()
	if width < 24 {
		width = 24
	}
	// Keep one spare column to avoid automatic wrap at the right edge.
	usable := width - 1
	line := fitConsoleLine(build(usable), usable)

	if r.active {
		rows := visualRows(r.lastText, width)
		clearActiveConsoleRows(rows)
	} else {
		fmt.Print("\r")
	}
	fmt.Print(line)
	r.lastText = line
	r.active = true
}

func (r *progressRenderer) Finish(build func(width int) string) {
	// Show the final state once, then erase the live row instead of committing a
	// long 100%% progress bar to terminal history. The caller prints a short
	// completion line immediately afterwards. This avoids Windows Terminal
	// reflowing historical progress bars when the window is made narrower.
	r.Render(build)
	r.Clear()
}

func (r *progressRenderer) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active {
		return
	}
	width := visibleConsoleWidth()
	if width < 24 {
		width = 24
	}
	clearActiveConsoleRows(visualRows(r.lastText, width))
	r.active = false
	r.lastText = ""
}

func visualRows(s string, width int) int {
	if width < 1 {
		return 1
	}
	n := utf8.RuneCountInString(s)
	if n <= 0 {
		return 1
	}
	rows := (n + width - 1) / width
	if rows < 1 {
		rows = 1
	}
	return rows
}

func fitConsoleLine(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	if max <= 3 {
		return strings.Repeat(".", max)
	}
	rr := []rune(s)
	return string(rr[:max-3]) + "..."
}
