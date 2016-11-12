package check

import "os/exec"

type ExecCheckExecutor struct {
	Command   *exec.Cmd
	Running   bool
	failed    bool
	lastError error
	message   string
}

func (e *ExecCheckExecutor) Start() {
	//TODO Implement a loop to kill the process if it's still running from the previous invocation
	if !e.Running {
		e.Running = true
		err := e.Command.Start()
		if err != nil {
			e.failed = true
		}
		e.Command.Wait()

		e.failed = !e.Command.ProcessState.Success()
		e.Running = false
	}
}

// Failed returns true if the last run of the check resulted in a failure
func (e *ExecCheckExecutor) Failed() bool {
	return e.failed
}

// LastError returns the error (if there was one) from the last run of the check
func (e *ExecCheckExecutor) LastError() error {
	return e.lastError
}

// Message returns the status information of the last check
func (e *ExecCheckExecutor) Message() string {
	return e.message
}
