//
// Sample monitor -- boilerplate for creating additional checks.
//
// Pretty simple:
//
// - In the New* constructor, create an instance of your monitor struct, assign
//   MonitorFunc to the actual, private "check" method; return struct instance.
// - Your check method should return an error or nil, depending on check status.
// - Implement a 'Validate' method that validates check-specific MonitorConfig bits
// - That's it - everything else is automatically handled by `Base` for you.
//

package monitor

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

func (s *SampleMonitor) Validate() error {
	return nil
}

func (s *SampleMonitor) sampleCheck() error {
	s.RMC.Log.WithField("configName", s.RMC.ConfigName).Debug("Performing sample check")

	return nil
}
