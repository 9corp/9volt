package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/alerter"
	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/state"
	"github.com/9corp/9volt/util"
)

const (
	START int = iota
	STOP

	GOROUTINE_ID_LENGTH = 8
	MAX_PORT            = 65536
)

type IMonitor interface {
	Run() error
	Stop()
	Identify() string
	Validate() error
}

type Monitor struct {
	Config             *config.Config
	Identifier         string
	runningMonitorLock *sync.Mutex
	runningMonitors    map[string]IMonitor
	MessageChannel     chan *alerter.Message
	StateChannel       chan *state.Message
	SupportedMonitors  map[string]func(*RootMonitorConfig) IMonitor // monitor name : NewXMonitor
	MemberID           string
}

type RootMonitorConfig struct {
	GID            string // goroutine id
	Name           string // monitor config name in member dir
	ConfigName     string // monitor config name in monitor dir
	MemberID       string
	Config         *MonitorConfig
	MessageChannel chan *alerter.Message
	StateChannel   chan *state.Message
	StopChannel    chan bool
	Ticker         *time.Ticker
}

// TODO: This should probably be split up between each individual check type
type MonitorConfig struct {
	// Generic attributes that fit more than one monitor type
	Type        string              `json:"type"`                  // 'tcp', 'http', 'ssh', 'exec', 'icmp', 'dns'
	Description string              `json:"description,omitempty"` // optional
	Host        string              `json:"host,omitempty"`        // required for all checks except 'exec'
	Interval    util.CustomDuration `json:"interval,omitempty"`
	Timeout     util.CustomDuration `json:"timeout,omitempty"`
	Port        int                 `json:"port,omitempty"`   // works for all checks except 'icmp' and 'exec'
	Expect      string              `json:"expect,omitempty"` // works for 'tcp', 'ssh', 'http', 'exec' checks except 'icmp'
	Disable     bool                `json:"disable,omitempty"`
	Tags        []string            `json:"tags,omitempty"`

	// TCP specific attributes
	TCPSend         string              `json:"send,omitempty"`
	TCPReadTimeout  util.CustomDuration `json:"read-timeout,omitempty"`
	TCPWriteTimeout util.CustomDuration `json:"write-timeout,omitempty"`
	TCPReadSize     int                 `json:"read-size,omitempty"`

	// HTTP specific attributes
	HTTPURL         string `json:"url,omitempty"`
	HTTPMethod      string `json:"method,omitempty"`
	HTTPSSL         bool   `json:"ssl,omitempty"`
	HTTPStatusCode  int    `json:"status-code,omitempty"`
	HTTPRequestBody string `json:"request-body,omitempty"` // Only used if 'Method' is 'GET'

	// Exec specific attributes
	ExecCommand    string   `json:"command,omitempty"`
	ExecArgs       []string `json:"args,omitempty"`
	ExecReturnCode int      `json:"return-code,omitempty"`

	// Alerting related configuration
	WarningThreshold  int      `json:"warning-threshold,omitempty"`  // how many times a check must fail before a warning alert is emitted
	CriticalThreshold int      `json:"critical-threshold,omitempty"` // how many times a check must fail before a critical alert is emitted
	WarningAlerter    []string `json:"warning-alerter,omitempty"`    // these alerters will be contacted when a warning threshold is hit
	CriticalAlerter   []string `json:"critical-alerter,omitempty"`   // these alerters will be contacted when a critical threshold is hit
}

type Response struct{}

func New(cfg *config.Config, messageChannel chan *alerter.Message, stateChannel chan *state.Message) *Monitor {
	return &Monitor{
		Identifier:     "monitor",
		Config:         cfg,
		MessageChannel: messageChannel,
		StateChannel:   stateChannel,
		MemberID:       cfg.MemberID,
		SupportedMonitors: map[string]func(*RootMonitorConfig) IMonitor{
			"http": NewHTTPMonitor,
			"tcp":  NewTCPMonitor,
			"exec": NewExecMonitor,
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
		m.Config.EQClient.AddWithErrorLog("error",
			fmt.Sprintf("%v: Unable to fetch monitor configuration for '%v' (%v): %v",
				m.Identifier, monitorName, monitorConfigLocation, err.Error()))
		return err
	}

	// validate monitor configuration
	if err := m.validateMonitorConfig(monitorConfig); err != nil {
		m.Config.EQClient.AddWithErrorLog("error",
			fmt.Sprintf("%v: Unable to validate monitor config for '%v' (%v): %v",
				m.Identifier, monitorName, monitorConfigLocation, err.Error()))
		return fmt.Errorf("Unable to validate monitor configuration for %v: %v", monitorName, err.Error())
	}

	// if check already running, stop it
	if m.monitorRunning(monitorName) {
		log.Debugf("%v: Monitor '%v' already running. Stopping it first...", m.Identifier, monitorName)

		if err := m.stop(monitorName); err != nil {
			m.Config.EQClient.AddWithErrorLog("error",
				fmt.Sprintf("%v: Unable to stop running monitor '%v': %v", m.Identifier, monitorName, err.Error()))
			return fmt.Errorf("Unable to stop running monitor %v: %v", monitorName, err.Error())
		}
	}

	// If check is disabled, do not start it back up
	if monitorConfig.Disable {
		log.Debugf("%v: '%v' is disabled. No further action will be taken.", m.Identifier, monitorName)
		return nil
	}

	// start check with new monitor configuration
	log.Debugf("%v: Starting new monitor for %v...", m.Identifier, monitorName)
	if err := m.start(monitorName, monitorConfigLocation, monitorConfig); err != nil {
		m.Config.EQClient.AddWithErrorLog("error",
			fmt.Sprintf("%v: Unable to start new monitor '%v': %v", m.Identifier, monitorName, err.Error()))
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
			ConfigName:     path.Base(monitorConfigLocation),
			GID:            util.RandomString(GOROUTINE_ID_LENGTH, false),
			Config:         monitorConfig,
			MemberID:       m.MemberID,
			MessageChannel: m.MessageChannel,
			StateChannel:   m.StateChannel,
			StopChannel:    make(chan bool, 1),
			Ticker:         time.NewTicker(time.Duration(monitorConfig.Interval)),
		},
	)

	// Do check-specific validation
	if err := newMonitor.Validate(); err != nil {
		return fmt.Errorf("%v: '%v' failed '%v' monitor config validation: %v",
			m.Identifier, path.Base(monitorConfigLocation), monitorConfig.Type, err.Error())
	}

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

	for k := range m.runningMonitors {
		if k == monitorName {
			return true
		}
	}

	return false
}

// Top level mnitor config validation
func (m *Monitor) validateMonitorConfig(monitorConfig *MonitorConfig) error {
	if monitorConfig.Interval.String() == "0s" {
		return errors.New("'interval' must be > 0s")
	}

	if monitorConfig.CriticalThreshold == 0 {
		return errors.New("'critical-threshold' must be non-zero")
	}

	// TODO: Logic for this should be changed/fixed at some point
	if monitorConfig.WarningThreshold > monitorConfig.CriticalThreshold {
		return errors.New("'warning-threshold' cannot be larger than 'CriticalThreshold'")
	}

	// TODO: It should be possible to NOT have a WarningAlerter setting (and just
	// have a `CriticalAlerter` setting)
	if len(monitorConfig.WarningAlerter) == 0 {
		return errors.New("'warning-alerter' list must contain at least one entry")
	}

	if len(monitorConfig.CriticalAlerter) == 0 {
		return errors.New("'critical-alerter' list must contain at least one entry")
	}

	if monitorConfig.Port > MAX_PORT {
		return fmt.Errorf("'port' must be between 0 and %v", MAX_PORT)
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
