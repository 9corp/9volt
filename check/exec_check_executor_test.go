package check

import (
	"os/exec"
	"testing"
)

func makeExecExecutor() ExecCheckExecutor {
	e := ExecCheckExecutor{}
	e.Command = exec.Command("echo", "hello")
	return e
}

func TestStart(t *testing.T) {
	e := makeExecExecutor()
	if e.Failed() != false {
		t.Errorf("Failed() was %t and it should have been false", e.Failed)
	}
}
