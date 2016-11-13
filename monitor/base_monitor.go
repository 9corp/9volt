package monitor

import (
	log "github.com/Sirupsen/logrus"
)

type Base struct {
	RMC         *RootMonitorConfig
	Identifier  string
	MonitorFunc func() error
}

func (b *Base) Stop() {
	b.RMC.StopChannel <- true
}

func (b *Base) Identify() string {
	return b.Identifier
}

// Actual run
func (b *Base) Run() error {
	log.Debugf("%v-%v: Starting work for monitor %v...", b.Identify(), b.RMC.GID, b.RMC.Name)

	defer b.RMC.Ticker.Stop()

Mainloop:
	for {
		select {
		case <-b.RMC.Ticker.C:
			log.Debugf("%v-%v: Tick for monitor %v", b.Identify(), b.RMC.GID, b.RMC.Name)
			b.MonitorFunc()
		case <-b.RMC.StopChannel:
			break Mainloop
		}
	}

	log.Warningf("%v-%v: Goroutine has been stopped for monitor %v; exiting...", b.Identify(), b.RMC.GID, b.RMC.Name)
	return nil
}
