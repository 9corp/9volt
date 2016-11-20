package alerter

import (
	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/config"
)

type Pagerduty struct {
	Config     *config.Config
	Identifier string
}

func NewPagerduty(cfg *config.Config) *Pagerduty {
	return &Pagerduty{
		Config:     cfg,
		Identifier: "pagerduty",
	}
}

func (p *Pagerduty) Send(msg *Message, alerterConfig *AlerterConfig) error {
	log.Debugf("%v: Sending message %v", p.Identifier, msg.uuid)
	return nil
}

func (p *Pagerduty) Identify() string {
	return p.Identifier
}
