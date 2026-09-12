//go:build !windows

package main

import (
	"fmt"
	"os"
	"strconv"
)

func initConsole() {}
func visibleConsoleWidth() int {
	if s := os.Getenv("COLUMNS"); s != "" {
		if n, e := strconv.Atoi(s); e == nil && n > 10 {
			return n
		}
	}
	return 100
}
func clearCurrentLine()   { fmt.Print("\r\x1b[2K") }
func clearConsoleScreen() { fmt.Print("\x1b[2J\x1b[3J\x1b[H") }

func protectConsoleCloseDuringScan() func() { return func() {} }
