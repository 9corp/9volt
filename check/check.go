package check

const (
	PendingState  = iota
	RunningState  = iota
	FailedState   = iota
	ResolvedState = iota
	ErrorState    = iota
)

type ICheck interface {
	StartCheck() error
	KillCheck() error
	CurrentState() int
	SetState(state int) error
	AddListener(state int, callback func(check ICheck)) error
	ClearListeners(state int) error
}
