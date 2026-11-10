//go:build windows

package process

import (
	"os"
	"os/exec"
	"syscall"
)

func prepareSys(cmd *exec.Cmd) {
	// CREATE_NEW_PROCESS_GROUP matches Unix Setpgid: Close can signal the child without a shell.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func interruptProcess(p *os.Process, sig os.Signal) error {
	if p == nil {
		return nil
	}
	return p.Signal(sig)
}

func killProcess(p *os.Process) error {
	if p == nil {
		return nil
	}
	return p.Kill()
}
