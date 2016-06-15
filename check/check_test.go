package check

import (
	"testing"
	"time"
)

func makeCheck() Check {
	return New("dummy", "", "")
}

func TestStartCheck(t *testing.T) {
	e := makeCheck()

	if e.StartCheck() != nil {
		t.Fail()
	}

	exec := e.Executor.(*DummyCheckExecutor)
	if !exec.Started {
		t.Error("Executor not started")
		t.Fail()
	}
}

func TestKillCheck(t *testing.T) {
	e := makeCheck()

	e.StartCheck()
	if e.KillCheck() != nil {
		t.Fail()
	}

	exec := e.Executor.(*DummyCheckExecutor)
	if !(exec.Started && exec.Stopped) {
		t.Errorf("Executor not started and stopped.\nStarted: %t\nStopped: %t\n",
			exec.Started, exec.Stopped)
		t.Fail()
	}
}

func TestCurrentState(t *testing.T) {
	e := makeCheck()

	if e.CurrentState() != PendingState {
		t.Fail()
	}
}

func TestSetState(t *testing.T) {
	e := makeCheck()

	e.SetState(RunningState)
	if e.CurrentState() != RunningState {
		t.Fail()
	}
}

func TestAddListener(t *testing.T) {
	e := makeCheck()

	failed := true
	e.AddListener(RunningState, func(check ICheck) {
		failed = false
	})

	e.SetState(RunningState)

	time.Sleep(2 * time.Millisecond)

	if failed {
		t.Fail()
	}
}

func TestClearListeners(t *testing.T) {
	e := makeCheck()

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
