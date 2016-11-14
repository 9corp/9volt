package monitor

import (
	"fmt"

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
		// Send critical resolve if critical threshold was exceed
		// Send warning resolve if warning threshold was exceeded but critical threshold was not exceeded
		if b.attemptCount > b.RMC.Config.CriticalThreshold {
			alertMessage := fmt.Sprintf("Check has resolved after %v/%v attempts (critical)", b.attemptCount, b.RMC.Config.CriticalThreshold)
			b.resolveAlert(CRITICAL, alertMessage)
		} else if b.attemptCount > b.RMC.Config.WarningThreshold {
			alertMessage := fmt.Sprintf("Check has resolved after %v/%v attempts (warning)", b.attemptCount, b.RMC.Config.WarningThreshold)
			b.resolveAlert(WARNING, alertMessage)
		}

		b.attemptCount = 0

		return nil
	}

	// Got an error; do we need to send any alerts?
	if b.criticalAlertSent {
		log.Debugf("Critical alert for %v already sent; skipping alerting", b.RMC.Name)
		return nil
	}

	if b.warningAlertSent {
		log.Debugf("Warning alert for %v already sent; skipping alerting", b.RMC.Name)
		return nil
	}

	// Okay, this must be the first time
	if b.attemptCount > b.RMC.Config.CriticalThreshold {
		alertMessage := fmt.Sprintf("Check has entered into warning state after %v checks (WarningThreshold: %v)", b.attemptCount, b.RMC.Config.CriticalThreshold)
		b.sendAlert(CRITICAL, alertMessage)
	} else if b.attemptCount > b.RMC.Config.WarningThreshold {
		alertMessage := fmt.Sprintf("Check has entered into critical state after %v checks (CriticalThreshold: %v)", b.attemptCount, b.RMC.Config.WarningThreshold)
		b.sendAlert(WARNING, alertMessage)
	}

	return nil
}

// Construct a new alert message, send down the message channel
func (b *Base) sendAlert(alertType int, alertMessage string) error {
	switch alertType {
	case CRITICAL:
		b.criticalAlertSent = true
	case WARNING:
		b.warningAlertSent = true
	}

	// Perform the alert send

	return nil
}

// Construct a new resolve alert message, send down the message channel
func (b *Base) resolveAlert(alertType int, resolveMessage string) error {
	switch alertType {
	case CRITICAL:
		b.criticalAlertSent = false
	case WARNING:
		b.warningAlertSent = false
	}

	// Perform the resolve alert send

	return nil
}
