//go:build darwin

package app

import (
	"syscall"
	"unsafe"
)

const (
	sysRenameatxNP = 488
	renameExcl     = 0x00000004
)

func renameNoReplace(oldPath, newPath string) error {
	oldPointer, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPointer, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}
	atFDCWD := ^uintptr(99)
	_, _, errno := syscall.Syscall6(sysRenameatxNP, atFDCWD, uintptr(unsafe.Pointer(oldPointer)), atFDCWD, uintptr(unsafe.Pointer(newPointer)), renameExcl, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
