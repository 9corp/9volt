package check

import (
	"testing"
)

func makeExecCheck() ExecCheck {
	return New("echo", "hello")
}

func TestStartCheck(t *testing.T) {
	e := makeExecCheck()

	if e.StartCheck() != nil {
		t.Fail()
	}
}

func TestKillCheck(t *testing.T) {
	e := New("/bin/sleep", "5")

	e.StartCheck()
	if e.KillCheck() != nil {
		t.Fail()
	}
}

func TestCurrentState(t *testing.T) {
	e := makeExecCheck()

	if e.CurrentState() != PendingState {
		t.Fail()
	}
}

func TestSetState(t *testing.T) {
	e := makeExecCheck()

	e.SetState(RunningState)
	if e.CurrentState() != RunningState {
		t.Fail()
	}
}

func TestAddListener(t *testing.T) {
	e := makeExecCheck()

	failed := true
	e.AddListener(RunningState, func(check ICheck) {
		failed = false
	})

	e.SetState(RunningState)

	if failed {
		t.Fail()
	}
}

func TestClearListeners(t *testing.T) {
	e := makeExecCheck()

	failed := false
	e.AddListener(RunningState, func(check ICheck) {
		failed = true
	})

	e.ClearListeners(RunningState)

	e.SetState(RunningState)

	if failed {
		t.Fail()
	}
}
