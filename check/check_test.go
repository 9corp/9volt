package check

import "testing"

func makeCheck() (Check, *CheckState) {
	c := New("dummy", "", "")
	state := new(CheckState)
	go func() {
		for {
			*state = <-c.UpdateChan
		}
	}()
	return c, state
}

func TestStartCheck(t *testing.T) {
	e, _ := makeCheck()

	if e.StartCheck() != nil {
		t.Fail()
	}

	exec := e.Executor.(*DummyCheckExecutor)
	if !exec.Started {
		t.Error("Executor not started")
		t.Fail()
	}
}

func TestCurrentState(t *testing.T) {
	e, _ := makeCheck()

	if e.CurrentState() != PendingState {
		t.Fail()
	}
}

func TestSetState(t *testing.T) {
	e, state := makeCheck()

	e.SetState(RunningState)
	if e.CurrentState() != RunningState {
		t.Fail()
	}

	if *state != RunningState {
		t.Fail()
	}
}
