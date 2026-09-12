//go:build windows

package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var procReadConsoleInputW = kernel32.NewProc("ReadConsoleInputW")

const (
	keyEventType = 0x0001
	vkBack       = 0x08
	vkReturn     = 0x0D
	vkEscape     = 0x1B
)

// readMenuInput reads ordinary menu input while making the physical Escape key
// an immediate, genuine "back" action in the Windows console. If standard input
// is redirected, it safely falls back to line input; typing "esc" then provides
// the same route for tests/scripts.
func readMenuInput(r *bufio.Reader) (string, bool) {
	h := syscall.Handle(os.Stdin.Fd())
	var mode uint32
	if ok, _, _ := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode))); ok == 0 {
		s := readLine(r)
		if strings.EqualFold(strings.TrimSpace(s), "esc") || s == "\x1b" {
			return "", true
		}
		return s, false
	}

	var runes []rune
	for {
		// INPUT_RECORD is 20 bytes on Windows. KEY_EVENT_RECORD begins at byte
		// offset 4 because the union is DWORD-aligned.
		var rec [20]byte
		var nread uint32
		ok, _, _ := procReadConsoleInputW.Call(
			uintptr(h),
			uintptr(unsafe.Pointer(&rec[0])),
			1,
			uintptr(unsafe.Pointer(&nread)),
		)
		if ok == 0 || nread == 0 {
			s := readLine(r)
			if strings.EqualFold(strings.TrimSpace(s), "esc") || s == "\x1b" {
				return "", true
			}
			return s, false
		}
		if binary.LittleEndian.Uint16(rec[0:2]) != keyEventType {
			continue
		}
		keyDown := binary.LittleEndian.Uint32(rec[4:8]) != 0
		if !keyDown {
			continue
		}
		vk := binary.LittleEndian.Uint16(rec[10:12])
		ch := binary.LittleEndian.Uint16(rec[14:16])

		switch vk {
		case vkEscape:
			fmt.Print("\r\n")
			return "", true
		case vkReturn:
			fmt.Print("\r\n")
			return strings.TrimSpace(string(runes)), false
		case vkBack:
			if len(runes) > 0 {
				runes = runes[:len(runes)-1]
				fmt.Print("\b \b")
			}
			continue
		}

		if ch < 0x20 {
			continue
		}
		rr := utf16.Decode([]uint16{ch})
		if len(rr) == 0 {
			continue
		}
		runes = append(runes, rr[0])
		fmt.Printf("%c", rr[0])
	}
}
