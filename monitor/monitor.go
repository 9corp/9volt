package monitor

import (
	"fmt"
	"sync"

	"github.com/9corp/9volt/config"
)

const (
	START int = iota
	STOP
)

type Monitor struct {
	Config             *config.Config
	Identifier         string
	runningMonitorLock *sync.Mutex
	runningMonitors    map[string]string
}

type MonitorConfig struct {
	Type        string
	Description string
	Tags        []string
}

func New(cfg *config.Config) *Monitor {
	return &Monitor{
		Identifier: "monitor",
		Config:     cfg,
	}
}

// Wrapper for handle(START, monitorConfig)
func (m *Monitor) Start(monitorName, monitorConfigLocation string) error {
	return m.handle(START, monitorName, monitorConfigLocation)
}

// Wrapper for
func (m *Monitor) Stop(monitorName, monitorConfigLocation string) error {
	return m.handle(STOP, monitorName, monitorConfigLocation)
}

// Start/stop or restart a monitor with a specific config
func (m *Monitor) handle(action int, monitorName, monitorConfigLocation string) error {
	// if stop action, check if we have a running instance of the check, if not, return an error
	if action == STOP {
		if m.monitorRunning(monitorName) {
			log.Debugf("%v: Stopping running monitor '%v'...", m.Identifier, monitorName)
			return m.stop(monitorName)
		}

		log.Errorf("%v: Asked to stop monitor '%v' but monitor is not running!", m.Identifier, monitorName)
		return fmt.Errorf("Asked to stop monitor %v but monitor is not running", monitorName)
	}

	// fetch fresh configuration from etcd
	monitorConfig, err := m.Config.DalClient.FetchMonitorConfig(monitorConfigLocation)
	if err != nil {
		log.Errorf("%v: Unable to fetch monitor configuration for '%v' (%v): %v",
			m.Identifier, monitorName, monitorConfigLocation, err.Error())
		return fmt.Errorf("Unable to fetch monitor configuration for %v: %v", monitorName, err.Error())
	}

	// validate monitor configuration
	if err := m.validateMonitorConfig(monitorConfig); err != nil {
		log.Errorf("%v: Unable to validate monitor config for '%v' (%v): %v",
			m.Identifier, monitorName, monitorConfigLocation, err.Error())
		return fmt.Errorf("Unable to validate monitor configuration for %v: %v", monitorName, err.Error())
	}

	// if check already running, stop it
	if m.monitorRunning(monitorName) {
		log.Debugf("%v: Monitor '%v' already running. Stopping it first...", m.Identifier, monitorName)

		if err := m.stop(monitorName); err != nil {
			log.Errorf("%v: Unable to stop running monitor '%v': %v", m.Identifier, monitorName, err.Error())
			return fmt.Errorf("Unable to stop running monitor %v: %v", monitorName, err.Error())
		}
	}

	// start check with new monitor configuration
	log.Debugf("%v: Starting new monitor for %v...", m.Identifier, monitorName)
	if err := m.start(monitorConfig); err != nil {
		log.Errorf("%v: Unable to start new monitor '%v': %v", m.Identifier, monitorName, err.Error())
		return fmt.Errorf("Unable to start new monitor %v: %v", monitorName, err.Error())
	}

	log.Debugf("%v: Successfully started new monitor %v!", m.Identifier, monitorName)

	return nil
}

// Perform the actual stop of a given monitor; update running monitor slice
func (m *Monitor) stop(monitorName string) error {
	// Stop the given monitor

	// Remove it from runningMonitors

	return nil
}

// Perform the actual start of a monitor; update running monitor slice
func (m *Monitor) start(monitorName string, monitorConfig *MonitorConfig) error {
	// Create a new monitor

	// Add monitor to runningMonitors

	return nil
}

// Determine if given `monitorName` is in `runningMonitors`
func (m *Monitor) monitorRunning(monitorName string) bool {
	m.runningMonitorLock.Lock()
	defer m.runningMonitorLock.Unlock()

	for k, v := range m.runningMonitors {
		if k == monitorName {
			return true
		}
	}

	return false
}
