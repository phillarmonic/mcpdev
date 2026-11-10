//go:build windows

package process

import "os"

func signalAlive(p *os.Process) error {
	return p.Signal(os.Kill)
}
