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
	Mutex        *sync.Mutex
	Data         map[string]*Message
	DumperLooper director.Looper

	base.Component
}

func New(cfg *config.Config, stateChannel chan *Message) *State {
	return &State{
		Config:       cfg,
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
	log.Infof("%v: Starting state components...", s.Identifier)

	s.Component.Ctx, s.Component.Cancel = context.WithCancel(context.Background())

	go s.runReader()
	go s.runDumper()

	return nil
}

func (s *State) Stop() error {
	log.Warningf("%v: Stopping all subcomponents", s.Identifier)

	if s.Component.Cancel == nil {
		log.Warningf("%v: Looks like .Cancel is nil; is this expected?", s.Identifier)
	} else {
		log.Warningf("YOYO: Called context for state")
		s.Component.Cancel()
	}

	// Shutdown dumper as well
	s.DumperLooper.Quit()

	return nil
}

// Read from state channel, update local state map; gets shutdown via context
func (s *State) runReader() error {
OUTER:
	for {
		select {
		case msg := <-s.StateChannel:

			// Safely write the message to the data map
			s.Mutex.Lock()
			s.Data[msg.Check] = msg
			s.Mutex.Unlock()

			log.Debugf("%v-runReader: Received state message for '%v'", s.Identifier, msg.Check)

			return nil
		case <-s.Component.Ctx.Done():
			log.Warningf("%v-runReader: Asked to shutdown", s.Identifier)
			break OUTER
		}
	}

	log.Warningf("%v-runReader: Exiting", s.Identifier)

	return nil
}

// Periodically dump state to etcd; gets shutdown via looper
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
				s.Config.EQClient.AddWithErrorLog("error",
					fmt.Sprintf("%v: Unable to marshal state message for key %v: %v", s.Identifier, k, err))
				continue
			}

			if err := s.Config.DalClient.Set(fullKey, string(messageBlob), nil); err != nil {
				s.Config.EQClient.AddWithErrorLog("error",
					fmt.Sprintf("%v: Unable to dump state for key %v: %v", s.Identifier, k, err))
				continue
			}

			delete(s.Data, k)
		}

		return nil
	})

	log.Warningf("%v-runDumper: Exiting", s.Identifier)

	return nil
}
