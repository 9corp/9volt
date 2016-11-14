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
	Name           string
	Config         *MonitorConfig
	MessageChannel chan *alerter.Message
	StopChannel    chan bool
	Ticker         *time.Ticker
}

// TODO: This should probably be split up between each individual check type
type MonitorConfig struct {
	// Generic attributes that fit more than one monitor type
	Type        string // 'tcp', 'http', 'ssh', 'exec', 'icmp', 'dns'
	Description string // optional
	Host        string // required for all checks except 'exec'
	Interval    util.CustomDuration
	Timeout     util.CustomDuration
	Port        int    // works for all checks except 'icmp' and 'exec'
	Expect      string // works for 'tcp', 'ssh', 'http', 'exec' checks except 'icmp'
	Enabled     bool
	Tags        []string

	// HTTP specific attributes
	HTTPURL         string
	HTTPMethod      string
	HTTPSSL         bool
	HTTPStatusCode  int
	HTTPRequestBody string // Only used if 'Method' is 'GET'

	// Exec specific attributes
	ExecCommand    string
	ExecReturnCode int

	// Alerting related configuration
	WarningThreshold  int      // how many times a check must fail before a warning alert is emitted
	CriticalThreshold int      // how many times a check must fail before a critical alert is emitted
	WarningAlerters   []string // these alerters will be contacted when a warning threshold is hit
	CriticalAlerters  []string // these alerters will be contacted when a critical threshold is hit
}

type Response struct{}

func New(cfg *config.Config, messageChannel chan *alerter.Message) *Monitor {
	return &Monitor{
		Identifier:     "monitor",
		Config:         cfg,
		MessageChannel: messageChannel,
		SupportedMonitors: map[string]func(*RootMonitorConfig) IMonitor{
			"http": NewHTTPMonitor,
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
	if err := m.start(monitorName, monitorConfig); err != nil {
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
func (m *Monitor) start(monitorName string, monitorConfig *MonitorConfig) error {
	// Let's be overly safe and ensure this monitor type exists
	if _, ok := m.SupportedMonitors[monitorConfig.Type]; !ok {
		return fmt.Errorf("%v: No such monitor type found '%v'", m.Identifier, monitorConfig.Type)
	}

	// Create a new monitor instance
	newMonitor := m.SupportedMonitors[monitorConfig.Type](
		&RootMonitorConfig{
			Name:           monitorName,
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
	// CriticalThreshold cannot be 0
	// WarningThreshold cannot be 0
	// WarningThreshold must be < CriticalThreshold (TODO: this should be updated with better logic)

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
