//go:build linux || freebsd

package exec

import "syscall"

func defSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}
}
