package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/9corp/9volt/alerter"
	"github.com/9corp/9volt/state"

	log "github.com/Sirupsen/logrus"
)

// States of a monitor
const (
	// OK when the alerts have resolved and everything is peachy
	OK int = iota
	// WARNING when the number of failed attempts passes the WarningThreshold
	WARNING
	// CRITICAL when the number of failed attempts passes the CriticalThreshold
	CRITICAL
)

var (
	okNextStates       = [2]int{WARNING, CRITICAL}
	warningNextStates  = [2]int{CRITICAL, OK}
	criticalNextStates = [2]int{WARNING, OK}
	stateTransition    = [3][2]int{okNextStates, warningNextStates, criticalNextStates}
)

// Base monitor to embed into monitors that do real work
type Base struct {
	RMC         *RootMonitorConfig
	Identifier  string
	MonitorFunc func() error

	attemptCount      int
	criticalAlertSent bool
	warningAlertSent  bool
	currentState      int
	resolveFuncs      map[string]func()
}

// Stop the monitor
func (b *Base) Stop() {
	b.RMC.StopChannel <- true
}

// Identify the monitor by a string
func (b *Base) Identify() string {
	return b.Identifier
}

// Run the check on a given interval -> evaluate response via b.handle()
func (b *Base) Run() error {
	log.Debugf("%v-%v: Starting work for monitor %v...", b.Identify(), b.RMC.GID, b.RMC.Name)

	defer b.RMC.Ticker.Stop()

	b.resolveFuncs = make(map[string]func())

Mainloop:
	for {
		select {
		case <-b.RMC.Ticker.C:
			log.Debugf("%v-%v: Tick for monitor %v", b.Identify(), b.RMC.GID, b.RMC.Name)
			if err := b.handle(b.MonitorFunc()); err != nil {
				log.Errorf("Unable to complete check handler: %v", err.Error())
			}
		case <-b.RMC.StopChannel:
			break Mainloop
		}
	}

	log.Debugf("%v-%v: Goroutine has been stopped for monitor %v; exiting...", b.Identify(), b.RMC.GID, b.RMC.Name)
	return nil
}

// Handle triggering/resolving alerts based on check results
func (b *Base) handle(monitorErr error) error {
	// Update state every run
	defer b.updateState(monitorErr)

	// No problems, reset counter
	if monitorErr == nil {
		b.transitionStateTo(OK, "")
		b.attemptCount = 0
		return nil
	}

	// Increase attempt count
	b.attemptCount++
	if b.attemptCount >= b.RMC.Config.CriticalThreshold {
		b.transitionStateTo(CRITICAL, monitorErr.Error())
	} else if b.attemptCount >= b.RMC.Config.WarningThreshold {
		b.transitionStateTo(WARNING, monitorErr.Error())
	}
	return nil
}

// Construct a new alert message, send down the message channel and update alert state
func (b *Base) sendMessage(curState int, titleMessage, alertMessage, errorDetails string, resolve bool) error {
	var alertType = [3]string{"resolve", "warning", "critical"}
	var alertKey = [3][]string{[]string{}, b.RMC.Config.WarningAlerter, b.RMC.Config.CriticalAlerter}

	log.Warningf("%v-%v: (%v) %v", b.Identifier, b.RMC.GID, b.RMC.Name, alertMessage)

	msg := &alerter.Message{
		Type:   alertType[curState],
		Key:    alertKey[curState],
		Title:  titleMessage,
		Text:   alertMessage,
		Count:  b.attemptCount,
		Source: b.RMC.ConfigName, // should be unique per check (used as incident key for PD)

		// Let's set some additional (potentially) useful info in the message
		Contents: map[string]string{
			"WarningThreshold":  fmt.Sprint(b.RMC.Config.WarningThreshold),
			"CriticalThreshold": fmt.Sprint(b.RMC.Config.CriticalThreshold),
			"ErrorDetails":      errorDetails,
		},
	}

	// Send the message
	b.RMC.MessageChannel <- msg

	log.Debugf("%v-%v: Successfully sent '%v' message for %v (%v)",
		b.Identifier, b.RMC.GID, msg.Type, b.RMC.ConfigName, b.RMC.Name)

	// Get resolve functions ready
	for _, alert := range alertKey[curState] {
		if _, exists := b.resolveFuncs[alert]; !exists {
			b.resolveFuncs[alert] = func() {
				msg := &alerter.Message{
					Type:   alertType[OK],
					Key:    []string{alert},
					Title:  titleMessage,
					Text:   alertMessage,
					Count:  b.attemptCount,
					Source: b.RMC.ConfigName, // should be unique per check (used as incident key for PD)

					// Let's set some additional (potentially) useful info in the message
					Contents: map[string]string{
						"WarningThreshold":  fmt.Sprint(b.RMC.Config.WarningThreshold),
						"CriticalThreshold": fmt.Sprint(b.RMC.Config.CriticalThreshold),
						"ErrorDetails":      errorDetails,
					},
				}

				// Send the message
				b.RMC.MessageChannel <- msg

				delete(b.resolveFuncs, alert)
			}
		}
	}

	return nil
}

// Construct a state message and send it down the state channel
//
// `updateState()` is intended to be ran *every* time `handle()` is ran; raw config
// is included for convenience.
func (b *Base) updateState(monitorErr error) error {
	var status = [3]string{"ok", "warning", "critical"}
	jsonConfig, err := json.Marshal(b.RMC.Config)
	if err != nil {
		errorMessage := fmt.Sprintf("Unable to marshal monitor config to JSON: %v", err.Error())
		jsonConfig = []byte(fmt.Sprintf(`{"error": "%v"}`, errorMessage))
		log.Error(errorMessage)
	}

	// If no error is set, set it to N/A for display purposes
	if monitorErr == nil {
		monitorErr = errors.New("N/A")
	}

	b.RMC.StateChannel <- &state.Message{
		Check:   b.RMC.ConfigName,
		Owner:   b.RMC.MemberID,
		Status:  status[b.currentState],
		Count:   b.attemptCount,
		Message: monitorErr.Error(),
		Date:    time.Now(),
		Config:  jsonConfig,
	}

	log.Debugf("%v: Successfully sent state message for '%v' to state reader", b.Identifier, b.RMC.ConfigName)

	return nil
}

func (b *Base) stateEvent(curState int, monitorErr string) {
	if curState == OK {
		for _, resolve := range b.resolveFuncs {
			resolve()
		}
		return
	}
	var stateStr = [3]string{"", "warning", "critical"}
	titleMessage := fmt.Sprintf("%v check '%v' failure", strings.ToUpper(b.Identify()), b.RMC.ConfigName)
	alertMessage := fmt.Sprintf("Check has entered into %s state after %v checks", stateStr[curState], b.attemptCount)
	b.sendMessage(curState, titleMessage, alertMessage, monitorErr, false)
}

func (b *Base) transitionStateTo(state int, monitorErr string) error {
	if state == b.currentState {
		return nil
	}
	for _, potentialNextState := range stateTransition[b.currentState] {
		if potentialNextState == state {
			b.currentState = state
			b.stateEvent(state, monitorErr)
			return nil
		}
	}
	return errors.New("State transition failed")
}
