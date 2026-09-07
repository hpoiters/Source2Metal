//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type coord struct{ X, Y int16 }
type smallRect struct{ Left, Top, Right, Bottom int16 }
type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleScreenBufInfo  = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procGetConsoleMode           = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode           = kernel32.NewProc("SetConsoleMode")
	procSetConsoleOutputCP       = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP             = kernel32.NewProc("SetConsoleCP")
	procFillConsoleOutputCharW   = kernel32.NewProc("FillConsoleOutputCharacterW")
	procFillConsoleOutputAttr    = kernel32.NewProc("FillConsoleOutputAttribute")
	procSetConsoleCursorPosition = kernel32.NewProc("SetConsoleCursorPosition")
)

const enableVirtualTerminalProcessing = 0x0004

func initConsole() {
	_, _, _ = procSetConsoleOutputCP.Call(65001)
	_, _, _ = procSetConsoleCP.Call(65001)
	h := syscall.Handle(os.Stdout.Fd())
	var mode uint32
	if r, _, _ := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode))); r != 0 {
		_, _, _ = procSetConsoleMode.Call(uintptr(h), uintptr(mode|enableVirtualTerminalProcessing))
	}
}

func visibleConsoleWidth() int {
	h := syscall.Handle(os.Stdout.Fd())
	var info consoleScreenBufferInfo
	if r, _, _ := procGetConsoleScreenBufInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&info))); r != 0 {
		w := int(info.Window.Right-info.Window.Left) + 1
		if w > 10 {
			return w
		}
	}
	return 100
}

func clearCurrentLine() {
	fmt.Print("\r\x1b[2K")
}

func clearConsoleScreen() {
	// Clear both the visible screen and Windows Terminal scrollback. The ANSI
	// clear is intentional: Windows Terminal can retain reflowed visual rows
	// outside the classic screen buffer after a resize.
	fmt.Print("\x1b[2J\x1b[3J\x1b[H")

	// Native fallback for classic conhost / VT-disabled situations.
	h := syscall.Handle(os.Stdout.Fd())
	var info consoleScreenBufferInfo
	if r, _, _ := procGetConsoleScreenBufInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&info))); r == 0 {
		return
	}
	cells := uint32(uint16(info.Size.X)) * uint32(uint16(info.Size.Y))
	var written uint32
	zero := uintptr(0)
	_, _, _ = procFillConsoleOutputCharW.Call(uintptr(h), uintptr(' '), uintptr(cells), zero, uintptr(unsafe.Pointer(&written)))
	_, _, _ = procFillConsoleOutputAttr.Call(uintptr(h), uintptr(info.Attributes), uintptr(cells), zero, uintptr(unsafe.Pointer(&written)))
	_, _, _ = procSetConsoleCursorPosition.Call(uintptr(h), zero)
}
