//
// Sample monitor -- boilerplate for creating additional checks.
//
// Pretty simple:
//
// - In the New* constructor, create an instance of your monitor struct, assign
//   MonitorFunc to the actual, private "check" method; return struct instance.
// - Your check method should return an error or nil, depending on check status.
// - That's it - everything else is automatically handled by `Base` for you.
//

package monitor

import (
	log "github.com/Sirupsen/logrus"
)

type SampleMonitor struct {
	Base
}

func NewSampleMonitor(rmc *RootMonitorConfig) IMonitor {
	s := &SampleMonitor{
		Base: Base{
			RMC:        rmc,
			Identifier: "sample",
		},
	}

	s.MonitorFunc = s.sampleCheck

	return s
}

func (s *SampleMonitor) sampleCheck() error {
	log.Debugf("%v-%v: Performing check for '%v'", s.Identifier, s.RMC.GID, s.RMC.ConfigName)

	return nil
}
