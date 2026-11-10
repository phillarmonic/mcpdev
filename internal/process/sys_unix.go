//go:build unix

package process

import (
	"os"
	"os/exec"
	"syscall"
)

func prepareSys(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func interruptProcess(p *os.Process, sig os.Signal) error {
	if p == nil {
		return nil
	}
	syssig, ok := sig.(syscall.Signal)
	if !ok {
		syssig = syscall.SIGTERM
	}
	return syscall.Kill(-p.Pid, syssig)
}

func killProcess(p *os.Process) error {
	if p == nil {
		return nil
	}
	return syscall.Kill(-p.Pid, syscall.SIGKILL)
}
