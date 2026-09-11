//go:build windows

package main

import "unsafe"

const processQueryLimitedInformation = 0x1000

type processIOCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

var (
	procOpenProcess          = kernel32.NewProc("OpenProcess")
	procCloseHandle          = kernel32.NewProc("CloseHandle")
	procGetProcessIoCounters = kernel32.NewProc("GetProcessIoCounters")
)

// processReadBytes returns Windows' process I/O accounting ReadTransferCount.
// It is used only as an optional live-progress hint. Exact checked bytes are
// still advanced only after tbcheck reports OK!/FAIL! for a file.
func processReadBytes(pid int) (uint64, bool) {
	h, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(uint32(pid)))
	if h == 0 {
		return 0, false
	}
	defer procCloseHandle.Call(h)
	var c processIOCounters
	r, _, _ := procGetProcessIoCounters.Call(h, uintptr(unsafe.Pointer(&c)))
	if r == 0 {
		return 0, false
	}
	return c.ReadTransferCount, true
}
