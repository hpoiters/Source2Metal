//go:build !windows

package main

func enableVirtualTerminal() bool { return false }

func consoleWidth() int { return 80 }
