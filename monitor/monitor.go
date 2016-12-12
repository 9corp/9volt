package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/alerter"
	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/util"
)

const (
	START int = iota
	STOP

	GOROUTINE_ID_LENGTH = 8
)

type IMonitor interface {
	Run() error
	Stop()
	Identify() string
}

type Monitor struct {
	Config             *config.Config
	Identifier         string
	runningMonitorLock *sync.Mutex
	runningMonitors    map[string]IMonitor
	MessageChannel     chan *alerter.Message
	SupportedMonitors  map[string]func(*RootMonitorConfig) IMonitor // monitor name : NewXMonitor
}

type RootMonitorConfig struct {
	GID            string // goroutine id
	Name           string // monitor config name in member dir
	Path           string // monitor config location in etcd
	Config         *MonitorConfig
	MessageChannel chan *alerter.Message
	StopChannel    chan bool
	Ticker         *time.Ticker
}

// TODO: This should probably be split up between each individual check type
type MonitorConfig struct {
	// Generic attributes that fit more than one monitor type
	Type        string              `json:"type"`        // 'tcp', 'http', 'ssh', 'exec', 'icmp', 'dns'
	Description string              `json:"description"` // optional
	Host        string              `json:"host"`        // required for all checks except 'exec'
	Interval    util.CustomDuration `json:"interval"`
	Timeout     util.CustomDuration `json:"timeout"`
	Port        int                 `json:"port"`   // works for all checks except 'icmp' and 'exec'
	Expect      string              `json:"expect"` // works for 'tcp', 'ssh', 'http', 'exec' checks except 'icmp'
	Disable     bool                `json:"disable"`
	Tags        []string            `json:"tags"`

	// TCP specific attributes
	TCPSend         string              `json:"send"`
	TCPReadTimeout  util.CustomDuration `json:"read-timeout"`
	TCPWriteTimeout util.CustomDuration `json:"write-timeout"`
	TCPReadSize     int                 `json:"read-size"`

	// HTTP specific attributes
	HTTPURL         string `json:"url"`
	HTTPMethod      string `json:"method"`
	HTTPSSL         bool   `json:"ssl"`
	HTTPStatusCode  int    `json:"status-code"`
	HTTPRequestBody string `json:"request-body"` // Only used if 'Method' is 'GET'

	// Exec specific attributes
	ExecCommand    string `json:"command"`
	ExecReturnCode int    `json:"return-code"`

	// Alerting related configuration
	WarningThreshold  int      `json:"warning-threshold"`  // how many times a check must fail before a warning alert is emitted
	CriticalThreshold int      `json:"critical-threshold"` // how many times a check must fail before a critical alert is emitted
	WarningAlerter    []string `json:"warning-alerter"`    // these alerters will be contacted when a warning threshold is hit
	CriticalAlerter   []string `json:"critical-alerter"`   // these alerters will be contacted when a critical threshold is hit
}

type Response struct{}

func New(cfg *config.Config, messageChannel chan *alerter.Message) *Monitor {
	return &Monitor{
		Identifier:     "monitor",
		Config:         cfg,
		MessageChannel: messageChannel,
		SupportedMonitors: map[string]func(*RootMonitorConfig) IMonitor{
			"http": NewHTTPMonitor,
			"tcp":  NewTCPMonitor,
		},
		runningMonitors:    make(map[string]IMonitor, 0),
		runningMonitorLock: &sync.Mutex{},
	}
}

// Start/stop or restart a monitor with a specific config
func (m *Monitor) Handle(action int, monitorName, monitorConfigLocation string) error {
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
	monitorConfig, err := m.fetchMonitorConfig(monitorConfigLocation)
	if err != nil {
		log.Errorf("%v: Unable to fetch monitor configuration for '%v' (%v): %v",
			m.Identifier, monitorName, monitorConfigLocation, err.Error())
		return err
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
	if err := m.start(monitorName, monitorConfigLocation, monitorConfig); err != nil {
		log.Errorf("%v: Unable to start new monitor '%v': %v", m.Identifier, monitorName, err.Error())
		return fmt.Errorf("Unable to start new monitor %v: %v", monitorName, err.Error())
	}

	log.Debugf("%v: Successfully started new monitor %v!", m.Identifier, monitorName)

	return nil
}

// Perform the actual stop of a given monitor; update running monitor slice
func (m *Monitor) stop(monitorName string) error {
	// Stop the given monitor
	m.runningMonitorLock.Lock()
	defer m.runningMonitorLock.Unlock()

	// Double check
	if _, ok := m.runningMonitors[monitorName]; !ok {
		return fmt.Errorf("%v: Unable to stop monitor '%v' - this monitor is NOT running!", m.Identifier, monitorName)
	}

	// Stop the actual check
	m.runningMonitors[monitorName].Stop()

	// Remove it from runningMonitors
	delete(m.runningMonitors, monitorName)

	return nil
}

// Perform the actual start of a monitor; update running monitor slice
func (m *Monitor) start(monitorName, monitorConfigLocation string, monitorConfig *MonitorConfig) error {
	// Let's be overly safe and ensure this monitor type exists
	if _, ok := m.SupportedMonitors[monitorConfig.Type]; !ok {
		return fmt.Errorf("%v: No such monitor type found '%v'", m.Identifier, monitorConfig.Type)
	}

	// Create a new monitor instance
	newMonitor := m.SupportedMonitors[monitorConfig.Type](
		&RootMonitorConfig{
			Name:           monitorName,
			Path:           monitorConfigLocation,
			GID:            util.RandomString(GOROUTINE_ID_LENGTH, false),
			Config:         monitorConfig,
			MessageChannel: m.MessageChannel,
			StopChannel:    make(chan bool, 1),
			Ticker:         time.NewTicker(time.Duration(monitorConfig.Interval)),
		},
	)

	m.runningMonitorLock.Lock()
	defer m.runningMonitorLock.Unlock()

	// Add monitor to runningMonitors
	m.runningMonitors[monitorName] = newMonitor

	// Launch the monitor
	go m.runningMonitors[monitorName].Run()

	return nil
}

// Determine if given `monitorName` is in `runningMonitors`
func (m *Monitor) monitorRunning(monitorName string) bool {
	m.runningMonitorLock.Lock()
	defer m.runningMonitorLock.Unlock()

	for k, _ := range m.runningMonitors {
		if k == monitorName {
			return true
		}
	}

	return false
}

// Ensure that the monitoring config is valid
func (m *Monitor) validateMonitorConfig(monitorConfig *MonitorConfig) error {
	// TODO: HTTP* validation

	if monitorConfig.Interval.String() == "0s" {
		return errors.New("'Interval' must be > 0s")
	}

	if monitorConfig.CriticalThreshold == 0 {
		return errors.New("'CriticalThreshold' must be non-zero")
	}

	// TODO: Logic for this should be changed/fixed at some point
	if monitorConfig.WarningThreshold > monitorConfig.CriticalThreshold {
		return errors.New("'WarningThreshold' cannot be larger than 'CriticalThreshold'")
	}

	// TODO: It should be possible to NOT have a WarningAlerter setting (and just
	// have a `CriticalAlerter` setting)
	if len(monitorConfig.WarningAlerter) == 0 {
		return errors.New("'WarningAlerter' list must contain at least one entry")
	}

	if len(monitorConfig.CriticalAlerter) == 0 {
		return errors.New("'CriticalAlerter' list must contain at least one entry")
	}

	return nil
}

// Wrapper for fetching (and unmarshaling) MonitorConfig by etcd location
func (m *Monitor) fetchMonitorConfig(monitorConfigLocation string) (*MonitorConfig, error) {
	monitorConfigData, err := m.Config.DalClient.Get(monitorConfigLocation, &dal.GetOptions{
		NoPrefix: true,
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch monitor configuration for '%v': %v", monitorConfigLocation, err.Error())
	}

	if _, ok := monitorConfigData[monitorConfigLocation]; !ok {
		return nil, errors.New("Returned monitor config data missing... bug?")
	}

	var monitorConfig *MonitorConfig

	if err := json.Unmarshal([]byte(monitorConfigData[monitorConfigLocation]), &monitorConfig); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal fetched monitorConfig for '%v': %v", monitorConfigLocation, err.Error())
	}

	return monitorConfig, nil
}
