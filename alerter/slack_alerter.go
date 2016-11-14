package alerter

import (
	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/config"
)

type Slack struct {
	Config     *config.Config
	Identifier string
}

func NewSlack(cfg *config.Config) *Slack {
	return &Slack{
		Config:     cfg,
		Identifier: "slack",
	}
}

func (s *Slack) Send(msg *Message, alerterConfig *AlerterConfig) error {
	log.Debugf("%v: Sending message %v...", s.Identifier, msg.uuid)
	return nil
}

func (s *Slack) Identify() string {
	return s.Identifier
}
