package monitor

import (
	log "github.com/Sirupsen/logrus"
)

type Base struct {
	RMC         *RootMonitorConfig
	Identifier  string
	MonitorFunc func() *Response
}

func (b *Base) Stop() {
	b.RMC.Ticker.Stop()
}

func (b *Base) Identify() string {
	return b.Identifier
}

// Actual run
func (b *Base) Run() error {
	log.Debugf("%v-%v: Starting work for monitor %v...", b.Identify(), b.RMC.GID, b.RMC.Name)

	for t := range b.RMC.Ticker.C {
		// execute the check
		log.Warningf("%v-%v: Tick at %v", b.Identify(), b.RMC.GID, t.String())

		b.MonitorFunc()
	}

	log.Debugf("%v-%v: Goroutine has been stopped for monitor %v; exiting...", b.Identify(), b.RMC.GID, b.RMC.Name)
	return nil
}
