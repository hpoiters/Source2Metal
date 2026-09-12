//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var folderIDDesktop = guid{0xB4BFCC3A, 0xDB2C, 0x424C, [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41}}

func desktopPath() (string, error) {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	ole32 := syscall.NewLazyDLL("ole32.dll")
	sh := shell32.NewProc("SHGetKnownFolderPath")
	free := ole32.NewProc("CoTaskMemFree")
	var p *uint16
	r, _, e := sh.Call(uintptr(unsafe.Pointer(&folderIDDesktop)), 0, 0, uintptr(unsafe.Pointer(&p)))
	if r == 0 && p != nil {
		defer free.Call(uintptr(unsafe.Pointer(p)))
		s := syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(p))[:])
		if s != "" {
			return s, nil
		}
	}
	if up := os.Getenv("USERPROFILE"); up != "" {
		return filepath.Join(up, "Desktop"), nil
	}
	return "", fmt.Errorf("Windows Desktop folder not found: %v", e)
}
