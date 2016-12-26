// Periodic check state -> etcd dumper

package state

import (
	"encoding/json"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/relistan/go-director"

	"github.com/9corp/9volt/config"
)

type Message struct {
	Check   string
	Owner   string
	Status  string
	Count   int
	Message string
	Date    time.Time
	Config  json.RawMessage
}

type State struct {
	Config       *config.Config
	StateChannel chan *Message
	Identifier   string

	ReaderLooper director.Looper
	DumperLooper director.Looper
}

func New(cfg *config.Config, stateChannel chan *Message) *State {
	return &State{
		Config:       cfg,
		StateChannel: stateChannel,
		Identifier:   "state",

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

		log.Debugf("%v: Received state message for '%v'", s.Identifier, msg.Check)

		return nil
	})

	return nil
}

// Periodically dump state to etcd
func (s *State) runDumper() error {
	s.DumperLooper.Loop(func() error {
		log.Debugf("%v: Dumping state to etcd every %v", s.Identifier, s.Config.StateDumpInterval.String())

		return nil
	})

	return nil
}
