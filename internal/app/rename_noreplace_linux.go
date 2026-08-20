//go:build linux

package app

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

const renameNoReplaceFlag = 1

func renameNoReplace(oldPath, newPath string) error {
	syscallNumber, err := renameat2SyscallNumber()
	if err != nil {
		return err
	}
	oldPointer, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPointer, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}
	atFDCWD := ^uintptr(99)
	_, _, errno := syscall.Syscall6(syscallNumber, atFDCWD, uintptr(unsafe.Pointer(oldPointer)), atFDCWD, uintptr(unsafe.Pointer(newPointer)), renameNoReplaceFlag, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func renameat2SyscallNumber() (uintptr, error) {
	// renameat2 is not exposed consistently by the frozen syscall package.
	// These numbers come from Linux's per-architecture unistd headers.
	byArchitecture := map[string]uintptr{
		"386":     353,
		"amd64":   316,
		"arm":     382,
		"arm64":   276,
		"loong64": 276,
		"ppc64":   357,
		"ppc64le": 357,
		"riscv64": 276,
		"s390x":   347,
	}
	number, found := byArchitecture[runtime.GOARCH]
	if !found {
		return 0, fmt.Errorf("atomic no-replace directory commit is unsupported on linux/%s", runtime.GOARCH)
	}
	return number, nil
}
