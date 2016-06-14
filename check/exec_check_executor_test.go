package check

import (
	"os/exec"
	"testing"
	"time"
)

func makeExecExecutor() ExecCheckExecutor {
	e := ExecCheckExecutor{}
	e.Command = exec.Command("echo", "hello")
	return e
}

func TestStart(t *testing.T) {
	e := makeExecExecutor()

	if e.Start() != nil && !e.Running {
		t.Fail()
	}
}

func TestStop(t *testing.T) {
	e := makeExecExecutor()
	e.Command = exec.Command("/bin/sleep", "30")

	e.Start()
	if e.Running && e.Stop() != nil {
		t.Error("Error when stopping")
	}

	// When dealing with processes, sometimes race conditions are a thing
	time.Sleep(10 * time.Millisecond)

	if e.Running {
		t.Error("process is still running")
	}
}
