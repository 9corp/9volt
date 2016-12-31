// Periodic check state -> etcd dumper

package state

import (
	"encoding/json"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/relistan/go-director"

	"github.com/9corp/9volt/config"
)

const (
	STATE_PREFIX = "state"
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
	StateChannel chan *Message
	Identifier   string
	Mutex        *sync.Mutex
	Data         map[string]*Message

	ReaderLooper director.Looper
	DumperLooper director.Looper
}

func New(cfg *config.Config, stateChannel chan *Message) *State {
	return &State{
		Config:       cfg,
		StateChannel: stateChannel,
		Identifier:   "state",
		Mutex:        &sync.Mutex{},
		Data:         make(map[string]*Message, 0),

		ReaderLooper: director.NewFreeLooper(director.FOREVER, make(chan error)),
		DumperLooper: director.NewTimedLooper(director.FOREVER, time.Duration(cfg.StateDumpInterval), make(chan error)),
	}
}

func (s *State) Start() error {
	log.Infof("%v: Starting state components...", s.Identifier)

	go s.runReader()
	go s.runDumper()

	return nil
}

// Read from state channel, update local state map
func (s *State) runReader() error {
	s.ReaderLooper.Loop(func() error {
		msg := <-s.StateChannel

		// Safely write the message to the data map
		s.Mutex.Lock()
		s.Data[msg.Check] = msg
		s.Mutex.Unlock()

		log.Debugf("%v: Received state message for '%v'", s.Identifier, msg.Check)

		return nil
	})

	return nil
}

// Periodically dump state to etcd
func (s *State) runDumper() error {
	s.DumperLooper.Loop(func() error {
		// log.Debugf("%v: Dumping state to etcd every %v", s.Identifier, s.Config.StateDumpInterval.String())

		s.Mutex.Lock()
		defer s.Mutex.Unlock()

		if len(s.Data) == 0 {
			// log.Debugf("%v: State is empty, nothing to do", s.Identifier)
			return nil
		}

		for k, v := range s.Data {
			fullKey := STATE_PREFIX + "/" + k

			messageBlob, err := json.Marshal(v)
			if err != nil {
				log.Errorf("%v: Unable to marshal state message for key %v: %v", s.Identifier, k, err)
				continue
			}

			if err := s.Config.DalClient.Set(fullKey, string(messageBlob), false, 0, ""); err != nil {
				log.Errorf("%v: Unable to dump state for key %v: %v", s.Identifier, k, err)
				continue
			}

			delete(s.Data, k)
		}

		return nil
	})

	return nil
}
