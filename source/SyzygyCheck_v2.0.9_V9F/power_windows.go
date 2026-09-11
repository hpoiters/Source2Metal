//go:build windows

package main

import "syscall"

const (
	esContinuous     = 0x80000000
	esSystemRequired = 0x00000001
)

func preventSystemSleep() func() {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("SetThreadExecutionState")
	_, _, _ = proc.Call(esContinuous | esSystemRequired)
	return func() { _, _, _ = proc.Call(esContinuous) }
}
