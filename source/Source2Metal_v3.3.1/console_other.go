//go:build !windows

package main

import (
	"fmt"
	"os"
	"strconv"
)

func visibleConsoleWidth() int {
	if s := os.Getenv("COLUMNS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return 100
}

func clearActiveConsoleRows(rows int) {
	if rows < 1 {
		rows = 1
	}
	fmt.Print("\r\x1b[2K")
	for i := 1; i < rows; i++ {
		fmt.Print("\x1b[1A\r\x1b[2K")
	}
	fmt.Print("\r")
}

func clearConsoleScreen() { fmt.Print("\x1b[2J\x1b[H") }
