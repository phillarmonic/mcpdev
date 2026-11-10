//go:build unix

package process

import (
	"os"
	"syscall"
)

func signalAlive(p *os.Process) error {
	return syscall.Kill(p.Pid, 0)
}
