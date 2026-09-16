//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

func enableVirtualTerminal() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")
	h := syscall.Handle(os.Stdout.Fd())
	var mode uint32
	r, _, _ := getMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return false
	}
	const enableVirtualTerminalProcessing = 0x0004
	r, _, _ = setMode.Call(uintptr(h), uintptr(mode|enableVirtualTerminalProcessing))
	return r != 0
}

// Query the actual console buffer width; wrapping follows the buffer width.
func consoleWidth() int {
	type coord struct{ X, Y int16 }
	type rect struct{ Left, Top, Right, Bottom int16 }
	type info struct {
		Size, Cursor coord
		Attributes   uint16
		Window       rect
		Maximum      coord
	}
	var value info
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleScreenBufferInfo")
	ok, _, _ := proc.Call(os.Stdout.Fd(), uintptr(unsafe.Pointer(&value)))
	if ok != 0 && value.Size.X > 1 {
		return int(value.Size.X)
	}
	return 80
}
