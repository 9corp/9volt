package alerter

import (
	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/config"
)

type Slack struct {
	Config     *config.Config
	identifier string
}

func NewSlack(cfg *config.Config) *Slack {
	return &Slack{
		Config:     cfg,
		identifier: "slack",
	}
}

func (s *Slack) Send(msg *Message, alerterConfig *AlerterConfig) error {
	log.Debugf("%v: Sending message %v...", s.identifier, msg.UUID)
	return nil
}

func (s *Slack) Identifier() string {
	return s.identifier
}
