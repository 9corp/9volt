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
	resolveMessages   map[string]*alerter.Message
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
	llog := b.RMC.Log.WithFields(log.Fields{"monitorName": b.RMC.Name, "method": b.RMC.Name})

	llog.Debug("Starting work")

	defer b.RMC.Ticker.Stop()

	b.resolveMessages = make(map[string]*alerter.Message)

Mainloop:
	for {
		select {
		case <-b.RMC.Ticker.C:
			llog.Debug("Monitor tick")
			if err := b.handle(b.MonitorFunc()); err != nil {
				log.Errorf("Unable to complete check handler: %v", err.Error())
			}
		case <-b.RMC.StopChannel:
			llog.Debug("Asked to shutdown")
			break Mainloop
		}
	}

	llog.Debug("Goroutine exiting...")
	return nil
}

// Handle triggering/resolving alerts based on check results
func (b *Base) handle(monitorErr error) error {
	var err error
	// Update state every run
	defer b.updateState(monitorErr)

	// No problems, reset counter
	if monitorErr == nil {
		err = b.transitionStateTo(OK, "")
		b.attemptCount = 0
		return nil
	}

	// Increase attempt count
	b.attemptCount++
	if b.attemptCount >= b.RMC.Config.CriticalThreshold {
		err = b.transitionStateTo(CRITICAL, monitorErr.Error())
	} else if b.attemptCount >= b.RMC.Config.WarningThreshold {
		err = b.transitionStateTo(WARNING, monitorErr.Error())
	}

	if err != nil {
		return err
	}
	return nil
}

// Construct a new alert message, send down the message channel and update alert state
func (b *Base) sendMessage(curState int, titleMessage, alertMessage, errorDetails string) error {
	var alertType = [3]string{"resolve", "warning", "critical"}
	var alertKey = [3][]string{[]string{}, b.RMC.Config.WarningAlerter, b.RMC.Config.CriticalAlerter}

	log.Debugf("%v-%v: (%v) %v", b.Identifier, b.RMC.GID, b.RMC.Name, alertMessage)

	msg := &alerter.Message{
		Type:        alertType[curState],
		Key:         alertKey[curState],
		Title:       titleMessage,
		Text:        alertMessage,
		Count:       b.attemptCount,
		Source:      b.RMC.ConfigName, // should be unique per check (used as incident key for PD)
		Description: b.RMC.Config.Description,

		// Let's set some additional (potentially) useful info in the message
		Contents: map[string]string{
			"WarningThreshold":  fmt.Sprint(b.RMC.Config.WarningThreshold),
			"CriticalThreshold": fmt.Sprint(b.RMC.Config.CriticalThreshold),
			"ErrorDetails":      errorDetails,
		},
	}

	// Send the message
	b.RMC.MessageChannel <- msg

	b.RMC.Log.WithFields(log.Fields{
		"configName": b.RMC.ConfigName,
		"msgType":    msg.Type,
		"name":       b.RMC.Name,
	}).Debug("Successfully sent message")

	// Get resolve functions ready
	for _, alert := range alertKey[curState] {
		// If we don't have a resolution message for the check then let's add it
		if _, exists := b.resolveMessages[alert]; !exists {
			resolvMsg := &alerter.Message{}
			// Copy the previous message
			*resolvMsg = *msg

			resolvMsg.Type = alertType[OK]
			resolvMsg.Key = []string{alert}

			b.resolveMessages[alert] = resolvMsg
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

	b.RMC.Log.WithField("configName", b.RMC.ConfigName).Debug("Successfully sent state message")

	return nil
}

func (b *Base) stateEvent(curState int, monitorErr string) {
	var stateStr = [3]string{"", "warning", "critical"}
	if curState == OK {
		for alert, resolve := range b.resolveMessages {
			// If we've resolved then let's send all those resolve messages
			resolve.Text = fmt.Sprintf("Check has recovered from %s after %v checks", stateStr[b.currentState], b.attemptCount)

			// Send the message
			b.RMC.MessageChannel <- resolve

			// Delete this call from the map
			delete(b.resolveMessages, alert)
		}
		return
	}
	titleMessage := fmt.Sprintf("%v check '%v' failure", strings.ToUpper(b.Identify()), b.RMC.ConfigName)
	alertMessage := fmt.Sprintf("Check has entered into %s state after %v checks", stateStr[curState], b.attemptCount)
	b.sendMessage(curState, titleMessage, alertMessage, monitorErr)
}

func (b *Base) transitionStateTo(state int, monitorErr string) error {
	// If the state is the same, then we don't want to trigger the events
	if state == b.currentState {
		return nil
	}

	for _, potentialNextState := range stateTransition[b.currentState] {
		// Is the state I want to transition to a valid next state
		if potentialNextState == state {
			b.stateEvent(state, monitorErr)
			b.currentState = state
			return nil
		}
	}
	return fmt.Errorf("Failed to transition from state %d to %d", b.currentState, state)
}

// setStateTransition is really only meant to be used in tests
func setStateTransition(idx int, transition [2]int) {
	stateTransition[idx] = transition
}
