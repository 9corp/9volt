// Periodic check state -> etcd dumper
package state

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/relistan/go-director"

	"github.com/9corp/9volt/base"
	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/util"
)

const (
	STATE_PREFIX         = "state"
	DEFAULT_STATE_TTL    = time.Hour * 24
	STATE_TTL_MULTIPLIER = 2
)

type Message struct {
	Check   string          `json:"check"`
	Owner   string          `json:"owner"`
	Status  string          `json:"status"`
	Count   int             `json:"count"`
	Message string          `json:"message"`
	Date    time.Time       `json:"date"`
	Config  json.RawMessage `json:"config"`
}

type State struct {
	Config       *config.Config
	Log          log.FieldLogger
	StateChannel chan *Message
	Mutex        *sync.Mutex
	Data         map[string]*Message
	DumperLooper director.Looper

	base.Component
}

// Used for parsing/fetching the interval from a config in a state message
type TmpMonitorConfig struct {
	Interval util.CustomDuration `json:"interval"`
}

func New(cfg *config.Config, stateChannel chan *Message) *State {
	return &State{
		Config:       cfg,
		Log:          log.WithField("pkg", "state"),
		StateChannel: stateChannel,
		Mutex:        &sync.Mutex{},
		Data:         make(map[string]*Message, 0),
		DumperLooper: director.NewTimedLooper(director.FOREVER, time.Duration(cfg.StateDumpInterval), make(chan error, 1)),
		Component: base.Component{
			Identifier: "state",
		},
	}
}

func (s *State) Start() error {
	s.Log.Info("Starting state components...")

	s.Component.Ctx, s.Component.Cancel = context.WithCancel(context.Background())

	go s.runReader()
	go s.runDumper()

	return nil
}

func (s *State) Stop() error {
	if s.Component.Cancel == nil {
		s.Log.Warning("Looks like .Cancel is nil; is this expected?")
	} else {
		s.Component.Cancel()
	}

	// Shutdown dumper as well
	s.DumperLooper.Quit()

	return nil
}

// Read from state channel, update local state map; gets shutdown via context
func (s *State) runReader() error {
	llog := s.Log.WithField("method", "runReader")

OUTER:
	for {
		select {
		case msg := <-s.StateChannel:

			// Safely write the message to the data map
			s.Mutex.Lock()
			s.Data[msg.Check] = msg
			s.Mutex.Unlock()

			llog.WithField("msg", msg.Check).Debug("Received state message")
		case <-s.Component.Ctx.Done():
			llog.Debug("Received a notice to shutdown")
			break OUTER
		}
	}

	llog.Debug("Exiting...")

	return nil
}

// Periodically dump state to etcd; gets shutdown via looper
func (s *State) runDumper() error {
	llog := s.Log.WithField("method", "runDumper")

	s.DumperLooper.Loop(func() error {
		// log.Debugf("%v: Dumping state to etcd every %v", s.Identifier, s.Config.StateDumpInterval.String())

		s.Mutex.Lock()
		defer s.Mutex.Unlock()

		if len(s.Data) == 0 {
			// log.Debugf("%v: State is empty, nothing to do", s.Identifier)
			return nil
		}

		for k, v := range s.Data {
			ttl, err := s.getInterval([]byte(v.Config))
			if err != nil {
				s.Config.EQClient.AddWithErrorLog("Unable to fetch interval", llog, log.Fields{"err": err})
				ttl = DEFAULT_STATE_TTL
			} else {
				// got a legitimate interval, let's increase it a bit
				ttl = ttl * STATE_TTL_MULTIPLIER
			}

			fullKey := STATE_PREFIX + "/" + k

			messageBlob, err := json.Marshal(v)
			if err != nil {
				s.Config.EQClient.AddWithErrorLog("Unable to marshal state message", llog, log.Fields{"key": k, "err": err})
				continue
			}

			if err := s.Config.DalClient.Set(fullKey, string(messageBlob), &dal.SetOptions{
				TTLSec: int(ttl.Seconds()),
			}); err != nil {
				s.Config.EQClient.AddWithErrorLog("Unable to dump state", llog, log.Fields{"key": k, "err": err})
				continue
			}

			delete(s.Data, k)
		}

		return nil
	})

	llog.Debug("Exiting...")

	return nil
}

// Fetch the run interval from a given json blob
func (s *State) getInterval(config []byte) (time.Duration, error) {
	var cfg TmpMonitorConfig

	if err := json.Unmarshal(config, &cfg); err != nil {
		return 0, fmt.Errorf("Unable to unmarshal config struct: %v", err)
	}

	return time.Duration(cfg.Interval), nil
}
