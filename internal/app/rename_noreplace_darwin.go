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

// renameNoReplaceAt renames oldName to newName relative to an open directory
// descriptor and fails with EEXIST rather than replacing an existing entry.
func renameNoReplaceAt(directory uintptr, oldName, newName string) error {
	oldPointer, err := syscall.BytePtrFromString(oldName)
	if err != nil {
		return err
	}
	newPointer, err := syscall.BytePtrFromString(newName)
	if err != nil {
		return err
	}
	_, _, errno := syscall.Syscall6(sysRenameatxNP, directory, uintptr(unsafe.Pointer(oldPointer)), directory, uintptr(unsafe.Pointer(newPointer)), renameExcl, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
