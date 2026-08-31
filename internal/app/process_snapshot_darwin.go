//go:build darwin && cgo

package app

/*
#include <libproc.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func nativeProcessIdentity(pid int) (processIdentity, bool, error) {
	var info C.struct_proc_bsdinfo
	got := C.proc_pidinfo(C.int(pid), C.PROC_PIDTBSDINFO, 0, unsafe.Pointer(&info), C.int(C.sizeof_struct_proc_bsdinfo))
	if got == 0 {
		return processIdentity{}, false, nil
	}
	if got != C.int(C.sizeof_struct_proc_bsdinfo) {
		return processIdentity{}, false, fmt.Errorf("read process identity for pid %d", pid)
	}
	return processIdentity{
		PID:       pid,
		ParentPID: int(info.pbi_ppid),
		Started:   fmt.Sprintf("%d.%06d", uint64(info.pbi_start_tvsec), uint64(info.pbi_start_tvusec)),
	}, true, nil
}

func childProcessIdentities(parentPID int) ([]processIdentity, error) {
	bytes := int(C.proc_listpids(C.PROC_PPID_ONLY, C.uint32_t(parentPID), nil, 0))
	if bytes <= 0 {
		return nil, nil
	}
	bytes += 32 * int(C.sizeof_int)
	buffer := C.malloc(C.size_t(bytes))
	if buffer == nil {
		return nil, fmt.Errorf("allocate child process identifier buffer")
	}
	defer C.free(buffer)
	written := int(C.proc_listpids(C.PROC_PPID_ONLY, C.uint32_t(parentPID), buffer, C.int(bytes)))
	if written < 0 {
		return nil, fmt.Errorf("read child process identifiers for pid %d", parentPID)
	}
	pids := unsafe.Slice((*C.int)(buffer), written/int(C.sizeof_int))
	result := make([]processIdentity, 0, len(pids))
	for _, rawPID := range pids {
		pid := int(rawPID)
		if pid <= 0 {
			continue
		}
		identity, found, err := nativeProcessIdentity(pid)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		result = append(result, identity)
	}
	return result, nil
}

func boundaryProcessIdentities(path string) ([]processIdentity, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	bytes := int(C.proc_listpidspath(C.PROC_ALL_PIDS, 0, cPath, 0, nil, 0))
	if bytes < 0 {
		return nil, fmt.Errorf("read process ownership boundary")
	}
	if bytes == 0 {
		return nil, nil
	}
	bytes += 32 * int(C.sizeof_int)
	buffer := C.malloc(C.size_t(bytes))
	if buffer == nil {
		return nil, fmt.Errorf("allocate process ownership boundary buffer")
	}
	defer C.free(buffer)
	written := int(C.proc_listpidspath(C.PROC_ALL_PIDS, 0, cPath, 0, buffer, C.int(bytes)))
	if written < 0 {
		return nil, fmt.Errorf("read process ownership boundary")
	}
	pids := unsafe.Slice((*C.int)(buffer), written/int(C.sizeof_int))
	result := make([]processIdentity, 0, len(pids))
	for _, rawPID := range pids {
		identity, found, err := nativeProcessIdentity(int(rawPID))
		if err != nil {
			return nil, err
		}
		if found {
			result = append(result, identity)
		}
	}
	return result, nil
}
