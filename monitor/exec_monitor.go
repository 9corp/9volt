package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	DEFAULT_CMD_TIMEOUT = time.Duration(3) * time.Second
)

type ExecMonitor struct {
	Base

	Timeout time.Duration
}

func NewExecMonitor(rmc *RootMonitorConfig) IMonitor {
	e := &ExecMonitor{
		Base: Base{
			RMC:        rmc,
			Identifier: "exec",
		},
	}

	// Set our cmd timeout
	if e.RMC.Config.Timeout.String() == "" {
		e.Timeout = DEFAULT_CMD_TIMEOUT
	} else {
		e.Timeout = time.Duration(e.RMC.Config.Timeout)
	}

	e.MonitorFunc = e.execCheck

	return e
}

func (e *ExecMonitor) execCheck() error {
	log.Debugf("%v-%v: Performing check for '%v'", e.Identifier, e.RMC.GID, e.RMC.ConfigName)

	ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
	defer cancel()

	// define, and rnu command
	output, err := exec.CommandContext(ctx, e.RMC.Config.ExecCommand, e.RMC.Config.ExecArgs...).CombinedOutput()
	if err != nil {
		// did the timeout?
		if err == context.DeadlineExceeded {
			return fmt.Errorf("Command execution hit timeout (%v)", e.RMC.Config.Timeout)
		}

		// It's possible that we are OK with a non-zero return code, so only bail
		// if our return code != expected return code in config
		if exiterr, ok := err.(*exec.ExitError); ok {
			// what return code did we get?
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if status.ExitStatus() != e.RMC.Config.ExecReturnCode {
					return fmt.Errorf("Command exited with '%v', expected '%v'. Output: %v",
						status.ExitStatus(), e.RMC.Config.ExecReturnCode, string(output))
				}
			} else {
				return fmt.Errorf("Unable to fetch return code status from command")
			}
		}
	}

	log.Warningf("EXEC: Err contents: %v Output: %v", err, string(output))

	if e.RMC.Config.Expect == "" {
		return nil
	}

	if !strings.Contains(string(output), e.RMC.Config.Expect) {
		return fmt.Errorf("Command output does not contain expected output. Expected: %v Output: %v",
			e.RMC.Config.Expect, string(output))
	}

	return nil
}
