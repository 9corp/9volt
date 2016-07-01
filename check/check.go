package check

const (
	PendingState = iota
	RunningState
	FailedState
	ResolvedState
	ErrorState
)

type ICheck interface {
	StartCheck() error
	KillCheck() error
	CurrentState() int
	SetState(state int) error
	AddListener(state int, callback func(check ICheck)) error
	ClearListeners(state int) error
}

type ICheckExecutor interface {
	Start() error
	Stop() error
}

type Check struct {
	state     int
	listeners map[int][]func(check ICheck)
	Executor  ICheckExecutor
}

func New(checkType string, command string, args string) Check {
	e := Check{}
	e.listeners = make(map[int][]func(check ICheck))
	switch checkType {
	case "exec":
		e.Executor = &ExecCheckExecutor{}
	case "http":
		e.Executor = &HTTPCheckExecutor{}
	default:
		e.Executor = &DummyCheckExecutor{}
	}
	return e
}

func (e *Check) StartCheck() error {
	return e.Executor.Start()
}

func (e *Check) KillCheck() error {
	return e.Executor.Stop()
}

func (e *Check) CurrentState() int {
	return e.state
}

func (e *Check) SetState(state int) error {
	for _, listener := range e.listeners[state] {
		go listener(e)
	}

	e.state = state

	return nil
}

func (e *Check) AddListener(state int, callback func(check ICheck)) error {
	e.listeners[state] = append(e.listeners[state], callback)
	return nil
}

func (e *Check) ClearListeners(state int) error {
	e.listeners[state] = []func(check ICheck){}
	return nil
}
