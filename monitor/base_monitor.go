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

const (
	WARNING int = iota
	CRITICAL
)

type Base struct {
	RMC         *RootMonitorConfig
	Identifier  string
	MonitorFunc func() error

	attemptCount      int
	criticalAlertSent bool
	warningAlertSent  bool
}

func (b *Base) Stop() {
	b.RMC.StopChannel <- true
}

func (b *Base) Identify() string {
	return b.Identifier
}

// Run the check on a given interval -> evaluate response via b.handle()
func (b *Base) Run() error {
	log.Debugf("%v-%v: Starting work for monitor %v...", b.Identify(), b.RMC.GID, b.RMC.Name)

	defer b.RMC.Ticker.Stop()

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

	// Increase attempt count
	b.attemptCount++

	// TCP check 'ssh-check' failure
	titleMessage := fmt.Sprintf("%v check '%v' failure", strings.ToUpper(b.Identify()), b.RMC.ConfigName)

	// No problems, reset counter
	if monitorErr == nil {
		// Send critical+warning resolve if critical threshold was exceed
		// Send warning resolve if warning threshold was exceeded but critical threshold was not exceeded
		if b.attemptCount > b.RMC.Config.CriticalThreshold {
			// Not digging any of this, but it's better than identical messages
			b.sendMessage(CRITICAL, titleMessage, fmt.Sprintf("Check has recovered from critical state after %v attempts", b.attemptCount), "", true)
			b.sendMessage(WARNING, titleMessage, fmt.Sprintf("Check has recovered from warning state after %v attempts", b.attemptCount), "", true)
		} else if b.attemptCount > b.RMC.Config.WarningThreshold {
			b.sendMessage(WARNING, titleMessage, fmt.Sprintf("Check has recovered from warning state after %v attempts", b.attemptCount), "", true)
		}

		b.attemptCount = 0

		return nil
	}

	// Got an error; do we need to send any alerts?
	if b.criticalAlertSent {
		log.Debugf("Critical alert for %v already sent; skipping alerting", b.RMC.Name)
		return nil
	}

	// Only return if we haven't reached critical threshold
	if b.warningAlertSent && b.attemptCount < b.RMC.Config.CriticalThreshold {
		log.Debugf("Warning alert for %v already sent; skipping alerting", b.RMC.Name)
		return nil
	}

	// Okay, this must be the first time
	if b.attemptCount == b.RMC.Config.CriticalThreshold {
		// TCP check 'some-check' failure! Host: some-host.com Port: N/A
		alertMessage := fmt.Sprintf("Check has entered into critical state after %v checks", b.attemptCount)
		b.sendMessage(CRITICAL, titleMessage, alertMessage, monitorErr.Error(), false)
	} else if b.attemptCount == b.RMC.Config.WarningThreshold {
		alertMessage := fmt.Sprintf("Check has entered into warning state after %v checks", b.attemptCount)
		b.sendMessage(WARNING, titleMessage, alertMessage, monitorErr.Error(), false)
	}

	return nil
}

// Construct a new alert message, send down the message channel and update alert state
func (b *Base) sendMessage(alertType int, titleMessage, alertMessage, errorDetails string, resolve bool) error {
	log.Warningf("%v-%v: (%v) %v", b.Identifier, b.RMC.GID, b.RMC.Name, alertMessage)

	msg := &alerter.Message{
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

	switch alertType {
	case CRITICAL:
		msg.Type = "critical"
		msg.Key = b.RMC.Config.CriticalAlerter

		// This is .. funky. To avoid having to set state in different places
		// and potentially requiring additional if/else||switch blocks, we set
		// the state to the reverse of the `resolve` bool
		b.criticalAlertSent = !resolve
	case WARNING:
		msg.Type = "warning"
		msg.Key = b.RMC.Config.WarningAlerter
		b.warningAlertSent = !resolve
	}

	if resolve {
		msg.Type = "resolve"
	}

	// Send the message
	b.RMC.MessageChannel <- msg

	log.Debugf("%v-%v: Successfully sent '%v' message for %v (%v)",
		b.Identifier, b.RMC.GID, msg.Type, b.RMC.ConfigName, b.RMC.Name)

	return nil
}

// Construct a state message and send it down the state channel
//
// `updateState()` is intended to be ran *every* time `handle()` is ran; raw config
// is included for convenience.
func (b *Base) updateState(monitorErr error) error {
	jsonConfig, err := json.Marshal(b.RMC.Config)
	if err != nil {
		errorMessage := fmt.Sprintf("Unable to marshal monitor config to JSON: %v", err.Error())
		jsonConfig = []byte(fmt.Sprintf(`{"error": "%v"}`, errorMessage))
		log.Error(errorMessage)
	}

	status := "ok"

	if b.criticalAlertSent {
		status = "critical"
	} else if b.warningAlertSent {
		status = "warning"
	}

	// If no error is set, set it to N/A for display purposes
	if monitorErr == nil {
		monitorErr = errors.New("N/A")
	}

	b.RMC.StateChannel <- &state.Message{
		Check:   b.RMC.ConfigName,
		Owner:   b.RMC.MemberID,
		Status:  status,
		Count:   b.attemptCount,
		Message: monitorErr.Error(),
		Date:    time.Now(),
		Config:  jsonConfig,
	}

	log.Debugf("%v: Successfully sent state message for '%v' to state reader", b.Identifier, b.RMC.ConfigName)

	return nil
}
