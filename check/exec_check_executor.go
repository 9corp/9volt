package check

import (
	"os/exec"
	"syscall"
)

type ExecCheckExecutor struct {
	Command *exec.Cmd
	Running bool
}

func (e *ExecCheckExecutor) Start() error {
	e.Running = true
	err := e.Command.Start()
	go func() {
		e.Command.Wait()
		e.Running = false
	}()
	return err
}

func (e *ExecCheckExecutor) Stop() error {
	//TODO: Escalate to SIGKILL if SIGTERM does nothing
	err := e.Command.Process.Signal(syscall.SIGTERM)
	return err
}
