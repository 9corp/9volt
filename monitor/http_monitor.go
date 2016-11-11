package monitor

import (
	log "github.com/Sirupsen/logrus"
)

type HTTPMonitor struct {
	Base
}

func NewHTTPMonitor(rmc *RootMonitorConfig) IMonitor {
	h := &HTTPMonitor{
		Base: Base{
			RMC:        rmc,
			identifier: "http",
		},
	}

	h.monitorFunc = h.httpCheck

	return h
}

func (h *HTTPMonitor) httpCheck() *Response {
	log.Debugf("%v-%v: Performing http check!", h.Identifier(), h.RMC.GID)
	return nil
}
