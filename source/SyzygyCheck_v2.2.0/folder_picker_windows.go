//go:build windows

package main

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var errFolderSelectionCancelled = errors.New("folder selection cancelled")

const (
	coinitApartmentThreaded = 0x2
	clsctxInprocServer      = 0x1
	fosPickFolders          = 0x20
	fosForceFileSystem      = 0x40
	fosPathMustExist        = 0x800
	sigdnFileSysPath        = 0x80058000
	hresultCancelled        = 0x800704C7
)

var (
	clsidFileOpenDialog = guid{0xDC1C5A9C, 0xE88A, 0x4DDE, [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileOpenDialog  = guid{0xD57C7288, 0xD4AD, 0x4768, [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
)

type iFileOpenDialog struct{ vtbl *iFileOpenDialogVtbl }
type iFileOpenDialogVtbl struct {
	queryInterface, addRef, release                                                     uintptr
	show, setFileTypes, setFileTypeIndex, getFileTypeIndex, advise, unadvise            uintptr
	setOptions, getOptions, setDefaultFolder, setFolder, getFolder, getCurrentSelection uintptr
	setFileName, getFileName, setTitle, setOKButtonLabel, setFileNameLabel, getResult   uintptr
}

type iShellItem struct{ vtbl *iShellItemVtbl }
type iShellItemVtbl struct {
	queryInterface, addRef, release, bindToHandler, getParent, getDisplayName uintptr
}

func hresultFailed(hr uintptr) bool { return int32(hr) < 0 }

func browseForFolder(title string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ole32 := syscall.NewLazyDLL("ole32.dll")
	coInitializeEx := ole32.NewProc("CoInitializeEx")
	coUninitialize := ole32.NewProc("CoUninitialize")
	coCreateInstance := ole32.NewProc("CoCreateInstance")
	coTaskMemFree := ole32.NewProc("CoTaskMemFree")

	hr, _, _ := coInitializeEx.Call(0, coinitApartmentThreaded)
	initialized := !hresultFailed(hr)
	if initialized {
		defer coUninitialize.Call()
	} else if uint32(hr) != 0x80010106 { // RPC_E_CHANGED_MODE
		return "", fmt.Errorf("CoInitializeEx failed: 0x%08X", uint32(hr))
	}

	var dialog *iFileOpenDialog
	hr, _, _ = coCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIFileOpenDialog)), uintptr(unsafe.Pointer(&dialog)),
	)
	if hresultFailed(hr) || dialog == nil {
		return "", fmt.Errorf("folder dialog could not be opened: 0x%08X", uint32(hr))
	}
	defer syscall.SyscallN(dialog.vtbl.release, uintptr(unsafe.Pointer(dialog)))

	hr, _, _ = syscall.SyscallN(dialog.vtbl.setOptions, uintptr(unsafe.Pointer(dialog)), fosPickFolders|fosForceFileSystem|fosPathMustExist)
	if hresultFailed(hr) {
		return "", fmt.Errorf("folder selection mode is unavailable: 0x%08X", uint32(hr))
	}
	if title16, err := syscall.UTF16PtrFromString(title); err == nil {
		syscall.SyscallN(dialog.vtbl.setTitle, uintptr(unsafe.Pointer(dialog)), uintptr(unsafe.Pointer(title16)))
	}

	hr, _, _ = syscall.SyscallN(dialog.vtbl.show, uintptr(unsafe.Pointer(dialog)), 0)
	if uint32(hr) == hresultCancelled {
		return "", errFolderSelectionCancelled
	}
	if hresultFailed(hr) {
		return "", fmt.Errorf("folder dialog failed: 0x%08X", uint32(hr))
	}

	var selected *iShellItem
	hr, _, _ = syscall.SyscallN(dialog.vtbl.getResult, uintptr(unsafe.Pointer(dialog)), uintptr(unsafe.Pointer(&selected)))
	if hresultFailed(hr) || selected == nil {
		return "", fmt.Errorf("selected folder is unavailable: 0x%08X", uint32(hr))
	}
	defer syscall.SyscallN(selected.vtbl.release, uintptr(unsafe.Pointer(selected)))

	var path16 *uint16
	hr, _, _ = syscall.SyscallN(selected.vtbl.getDisplayName, uintptr(unsafe.Pointer(selected)), sigdnFileSysPath, uintptr(unsafe.Pointer(&path16)))
	if hresultFailed(hr) || path16 == nil {
		return "", fmt.Errorf("selected folder path is unavailable: 0x%08X", uint32(hr))
	}
	defer coTaskMemFree.Call(uintptr(unsafe.Pointer(path16)))
	path := syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(path16))[:])
	if path == "" {
		return "", fmt.Errorf("selected folder path is empty")
	}
	return path, nil
}

func isFolderSelectionCancelled(err error) bool {
	return errors.Is(err, errFolderSelectionCancelled)
}
