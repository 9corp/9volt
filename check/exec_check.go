package check

import (
	"os/exec"
)

type ExecCheck struct {
	state     int
	listeners map[int][]func(check ICheck)
	command   *exec.Cmd
}

func New(command string, args string) ExecCheck {
	e := ExecCheck{}
	e.listeners = make(map[int][]func(check ICheck))
	e.command = exec.Command(command, args)
	return e
}

func (e *ExecCheck) StartCheck() error {
	err := e.command.Start()
	return err
}

func (e *ExecCheck) KillCheck() error {
	err := e.command.Process.Kill()
	return err
}

func (e *ExecCheck) CurrentState() int {
	return e.state
}

func (e *ExecCheck) SetState(state int) error {
	for _, listener := range e.listeners[state] {
		listener(e)
	}

	e.state = state

	return nil
}

func (e *ExecCheck) AddListener(state int, callback func(check ICheck)) error {
	e.listeners[state] = append(e.listeners[state], callback)
	return nil
}

func (e *ExecCheck) ClearListeners(state int) error {
	e.listeners[state] = []func(check ICheck){}
	return nil
}
