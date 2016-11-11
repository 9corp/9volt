package monitor

import (
	log "github.com/Sirupsen/logrus"
)

type Base struct {
	RMC         *RootMonitorConfig
	identifier  string
	monitorFunc func() *Response
}

func (b *Base) Stop() {
	b.RMC.Ticker.Stop()
}

func (b *Base) Identifier() string {
	return b.identifier
}

func (b *Base) Run() error {
	log.Debugf("%v-%v: Starting work for monitor %v...", b.Identifier(), b.RMC.GID, b.RMC.Name)

	for t := range b.RMC.Ticker.C {
		// execute the check
		log.Warningf("%v-%v: Tick at %v", b.Identifier(), b.RMC.GID, t.String())

		b.monitorFunc()
	}

	log.Debugf("%v-%v: Goroutine has been stopped for monitor %v; exiting...", b.Identifier(), b.RMC.GID, b.RMC.Name)
	return nil
}
