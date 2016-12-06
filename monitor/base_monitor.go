package monitor

import (
	"fmt"

	"github.com/9corp/9volt/alerter"

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

	log.Warningf("%v-%v: Goroutine has been stopped for monitor %v; exiting...", b.Identify(), b.RMC.GID, b.RMC.Name)
	return nil
}

// Handle triggering/resolving alerts based on check results
func (b *Base) handle(monitorErr error) error {
	// Increase attempt count
	b.attemptCount++

	// No problems, reset counter
	if monitorErr == nil {
		// Send critical+warning resolve if critical threshold was exceed
		// Send warning resolve if warning threshold was exceeded but critical threshold was not exceeded
		if b.attemptCount > b.RMC.Config.CriticalThreshold {
			alertMessage := fmt.Sprintf("Check has resolved after %v/%v attempts (critical)", b.attemptCount, b.RMC.Config.CriticalThreshold)
			b.sendMessage(CRITICAL, alertMessage, true)
			b.sendMessage(WARNING, alertMessage, true)
		} else if b.attemptCount > b.RMC.Config.WarningThreshold {
			alertMessage := fmt.Sprintf("Check has resolved after %v/%v attempts (warning)", b.attemptCount, b.RMC.Config.WarningThreshold)
			b.sendMessage(WARNING, alertMessage, true)
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
		alertMessage := fmt.Sprintf("Check has entered into critical state after %v checks (CriticalThreshold: %v)", b.attemptCount, b.RMC.Config.CriticalThreshold)
		b.sendMessage(CRITICAL, alertMessage, false)
	} else if b.attemptCount == b.RMC.Config.WarningThreshold {
		alertMessage := fmt.Sprintf("Check has entered into warning state after %v checks (WarningThreshold: %v)", b.attemptCount, b.RMC.Config.WarningThreshold)
		b.sendMessage(WARNING, alertMessage, false)
	}

	return nil
}

// Construct a new alert message, send down the message channel and update alert state
func (b *Base) sendMessage(alertType int, alertMessage string, resolve bool) error {
	log.Warningf("%v-%v: (%v) %v", b.Identifier, b.RMC.GID, b.RMC.Name, alertMessage)

	var alertTypeString string

	messageType := "alert"

	if resolve {
		messageType = "resolve"
	}

	msg := &alerter.Message{
		Text:   alertMessage,
		Count:  b.attemptCount,
		Source: b.Identifier,

		// Let's set some additional (potentially) useful info in the message
		Contents: map[string]string{
			"WarningThreshold":  fmt.Sprint(b.RMC.Config.WarningThreshold),
			"CriticalThreshold": fmt.Sprint(b.RMC.Config.CriticalThreshold),
		},

		Resolve: resolve,
	}

	switch alertType {
	case CRITICAL:
		msg.Critical = true
		msg.Key = b.RMC.Config.CriticalAlerter
		alertTypeString = "critical"

		// This is .. funky. To avoid having to set state in different places
		// and potentially requiring additional if/else||switch blocks, we set
		// the state to the reverse of the `resolve` bool
		b.criticalAlertSent = !resolve
	case WARNING:
		msg.Warning = true
		msg.Key = b.RMC.Config.WarningAlerter
		alertTypeString = "warning"
		b.warningAlertSent = !resolve
	}

	// Send the message
	b.RMC.MessageChannel <- msg

	log.Debugf("%v-%v: Successfully sent '%v' message (type: %v) for %v",
		b.Identifier, b.RMC.GID, messageType, alertTypeString, b.RMC.Name)

	return nil
}
