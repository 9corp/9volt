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
			Identifier: "http",
		},
	}

	h.MonitorFunc = h.httpCheck

	return h
}

func (h *HTTPMonitor) httpCheck() *Response {
	log.Debugf("%v-%v: Performing http check!", h.Identify(), h.RMC.GID)
	return nil
}
