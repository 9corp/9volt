package monitor

import (
	"context"
	"errors"
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
	FullCmd string
}

func NewExecMonitor(rmc *RootMonitorConfig) *ExecMonitor {
	e := &ExecMonitor{
		Base: Base{
			RMC:        rmc,
			Identifier: "exec",
		},
	}

	// Set our cmd timeout
	if e.RMC.Config.Timeout.String() == "0s" {
		e.Timeout = DEFAULT_CMD_TIMEOUT
	} else {
		e.Timeout = time.Duration(e.RMC.Config.Timeout)
	}

	e.FullCmd = e.getFullCmd()
	e.MonitorFunc = e.execCheck

	return e
}

func (e *ExecMonitor) Validate() error {
	log.Debugf("%v: Performing monitor config validation for %v", e.Identifier, e.RMC.ConfigName)

	if e.RMC.Config.ExecCommand == "" {
		return errors.New("'command' cannot be blank")
	}

	if e.Timeout >= time.Duration(e.RMC.Config.Interval) {
		return fmt.Errorf("'timeout' (%v) cannot equal or exceed 'interval' (%v)", e.Timeout.String(), e.RMC.Config.Interval.String())
	}

	return nil
}

func (e *ExecMonitor) execCheck() error {
	log.Debugf("%v-%v: Performing check for '%v'", e.Identifier, e.RMC.GID, e.RMC.ConfigName)

	ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
	defer cancel()

	// TODO: This could use a refactor at some point
	output, err := exec.CommandContext(ctx, e.RMC.Config.ExecCommand, e.RMC.Config.ExecArgs...).CombinedOutput()
	if err != nil {
		// Did we timeout?
		if err.Error() == "signal: killed" {
			return fmt.Errorf("Command '%v' exceeded run timeout (%v)", e.FullCmd, e.RMC.Config.Timeout.String())
		}

		// Did we exit non-zero?
		if exiterr, ok := err.(*exec.ExitError); ok {
			// what return code did we get?
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if status.ExitStatus() != e.RMC.Config.ExecReturnCode {
					return fmt.Errorf("Command '%v' exited with '%v', expected '%v'. Output: %v",
						e.FullCmd, status.ExitStatus(), e.RMC.Config.ExecReturnCode,
						strings.Replace(string(output), "\n", "\\n", -1))
				}
			} else {
				return fmt.Errorf("Unable to fetch return code for command '%v'. Full error: %v", e.FullCmd, err.Error())
			}
		} else {
			// Something else went bad
			return fmt.Errorf("Unexpected error during command '%v' execution: %v", e.FullCmd, err.Error())
		}
	} else {
		// Got a 0 return code; let's verify that we're okay with 0
		if e.RMC.Config.ExecReturnCode != 0 {
			return fmt.Errorf("Command '%v' exited with a '0' return code, expected '%v'", e.FullCmd, e.RMC.Config.ExecReturnCode)
		}
	}

	if e.RMC.Config.Expect == "" {
		return nil
	}

	if !strings.Contains(string(output), e.RMC.Config.Expect) {
		return fmt.Errorf("Command '%v' output does not contain expected output. Expected: %v Output: %v",
			e.FullCmd, e.RMC.Config.Expect, strings.Replace(string(output), "\n", "\\n", -1))
	}

	return nil
}

// Helper for displaying full cmd
func (e *ExecMonitor) getFullCmd() string {
	fullCmd := e.RMC.Config.ExecCommand

	for _, v := range e.RMC.Config.ExecArgs {
		fullCmd = fullCmd + " " + v
	}

	return fullCmd
}
