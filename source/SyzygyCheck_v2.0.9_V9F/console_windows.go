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
	procGetConsoleWindow         = kernel32.NewProc("GetConsoleWindow")
	user32                       = syscall.NewLazyDLL("user32.dll")
	procGetSystemMenu            = user32.NewProc("GetSystemMenu")
	procEnableMenuItem           = user32.NewProc("EnableMenuItem")
	procDrawMenuBar              = user32.NewProc("DrawMenuBar")
)

const (
	enableVirtualTerminalProcessing = 0x0004
	scClose                         = 0xF060
	mfByCommand                     = 0x0000
	mfGrayed                        = 0x0001
	mfDisabled                      = 0x0002
)

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
func clearCurrentLine() { fmt.Print("\r\x1b[2K") }
func clearConsoleScreen() {
	fmt.Print("\x1b[2J\x1b[3J\x1b[H")
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

// protectConsoleCloseDuringScan disables only the window Close command while a
// real Syzygy scan is active. Minimize, maximize, resizing and the scrollbar
// remain untouched. The returned function restores the previous Close state.
func protectConsoleCloseDuringScan() func() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return func() {}
	}
	menu, _, _ := procGetSystemMenu.Call(hwnd, 0)
	if menu == 0 {
		return func() {}
	}
	previous, _, _ := procEnableMenuItem.Call(menu, scClose, mfByCommand|mfGrayed)
	if previous == uintptr(0xFFFFFFFF) {
		return func() {}
	}
	_, _, _ = procDrawMenuBar.Call(hwnd)

	return func() {
		restoreState := previous & (mfGrayed | mfDisabled)
		_, _, _ = procEnableMenuItem.Call(menu, scClose, mfByCommand|restoreState)
		_, _, _ = procDrawMenuBar.Call(hwnd)
	}
}
