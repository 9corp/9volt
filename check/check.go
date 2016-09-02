package check

import (
	"os/exec"
)

const (
	PendingState = iota
	RunningState
	FailedState
	ErrorState
	SuccessState
)

type CheckState int

type ICheck interface {
	StartCheck() error
	CurrentState() CheckState
	SetState(state CheckState)
}

type ICheckExecutor interface {
	Start()
	Failed() bool
	LastError() error
}

type Check struct {
	state      CheckState
	Executor   ICheckExecutor
	UpdateChan chan CheckState
}

func New(checkType string, args ...string) Check {
	e := Check{
		UpdateChan: make(chan CheckState),
	}
	switch checkType {
	case "exec":
		e.Executor = &ExecCheckExecutor{
			Command: exec.Command(args[0], args[1:]...),
		}
	case "http":
		e.Executor = &HTTPCheckExecutor{
			URL: args[0],
		}
	default:
		e.Executor = &DummyCheckExecutor{}
	}
	return e
}

func (e *Check) StartCheck() error {
	e.SetState(RunningState)
	e.Executor.Start()

	if e.Executor.LastError() != nil {
		e.SetState(ErrorState)
		return e.Executor.LastError()
	}
	if e.Executor.Failed() {
		e.SetState(FailedState)
	} else {
		e.SetState(SuccessState)
	}

	return nil
}

func (e *Check) CurrentState() CheckState {
	return e.state
}

func (e *Check) SetState(state CheckState) {
	e.state = state
	e.UpdateChan <- state
}
