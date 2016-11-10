package monitor

import (
	log "github.com/Sirupsen/logrus"
)

type HTTPMonitor struct {
	RMC        *RootMonitorConfig
	identifier string
}

func NewHTTPMonitor(rmc *RootMonitorConfig) IMonitor {
	return &HTTPMonitor{
		RMC:        rmc,
		identifier: "http",
	}
}

func (h *HTTPMonitor) Run() error {
	log.Debugf("%v-%v: Starting work for monitor %v...", h.Identifier(), h.RMC.GID, h.RMC.Name)

	for t := range h.RMC.Ticker.C {
		// execute the check
		log.Warningf("%v-%v: Tick at %v", h.Identifier(), h.RMC.GID, t.String())

		h.httpCheck()
	}

	log.Debugf("%v-%v: Goroutine has been stopped for monitor %v; exiting...", h.Identifier(), h.RMC.GID, h.RMC.Name)
	return nil
}

func (h *HTTPMonitor) httpCheck() *Response {
	log.Debugf("%v-%v: Performing http check!", h.Identifier(), h.RMC.GID)
	return nil
}

func (h *HTTPMonitor) Identifier() string {
	return h.identifier
}

func (h *HTTPMonitor) Stop() {
	h.RMC.Ticker.Stop()
}
