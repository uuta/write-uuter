//go:build !darwin && !linux

package app

import "fmt"

func renameNoReplaceAt(_ uintptr, _, _ string) error {
	return fmt.Errorf("atomic no-replace directory rename is unsupported on this platform")
}
