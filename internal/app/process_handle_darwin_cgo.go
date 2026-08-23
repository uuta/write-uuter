//go:build darwin && cgo

package app

/*
#include <errno.h>
#include <libproc.h>
#include <mach/mach.h>
#include <mach/task_info.h>

static int write_uuter_audit_token(pid_t pid, audit_token_t *token) {
	mach_port_t task = MACH_PORT_NULL;
	kern_return_t result = task_name_for_pid(mach_task_self(), pid, &task);
	if (result != KERN_SUCCESS) {
		return (int)result;
	}
	mach_msg_type_number_t count = TASK_AUDIT_TOKEN_COUNT;
	result = task_info(task, TASK_AUDIT_TOKEN, (task_info_t)token, &count);
	mach_port_deallocate(mach_task_self(), task);
	return result == KERN_SUCCESS ? 0 : (int)result;
}

static int write_uuter_signal(audit_token_t *token, int signal) {
	return proc_signal_with_audittoken(token, signal) == 0 ? 0 : errno;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"syscall"
)

type stableProcess struct {
	identity processIdentity
	token    C.audit_token_t
}

var stableSignalTestHook func()
var stableAcquireTestHook func()

func openStableProcess(identity processIdentity) (*stableProcess, error) {
	matches, err := identityMatches(identity)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, errStaleProcessIdentity
	}
	process := &stableProcess{identity: identity}
	if stableAcquireTestHook != nil {
		stableAcquireTestHook()
	}
	if result := C.write_uuter_audit_token(C.pid_t(identity.PID), &process.token); result != 0 {
		matches, recheckErr := identityMatches(identity)
		if recheckErr != nil {
			return nil, recheckErr
		}
		if !matches {
			return nil, errStaleProcessIdentity
		}
		return nil, fmt.Errorf("acquire stable audit token for pid %d: mach error %d", identity.PID, int(result))
	}
	matches, err = identityMatches(identity)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, errStaleProcessIdentity
	}
	return process, nil
}

func (process *stableProcess) signal(signal syscall.Signal) error {
	if stableSignalTestHook != nil {
		stableSignalTestHook()
	}
	if result := C.write_uuter_signal(&process.token, C.int(signal)); result != 0 {
		err := syscall.Errno(result)
		if errors.Is(err, syscall.ESRCH) {
			return errStaleProcessIdentity
		}
		return fmt.Errorf("signal stable audit token for pid %d: %w", process.identity.PID, err)
	}
	return nil
}

func (process *stableProcess) close() error { return nil }
