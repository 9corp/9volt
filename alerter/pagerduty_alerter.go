package alerter

import (
	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/config"
)

type Pagerduty struct {
	Config     *config.Config
	identifier string
}

func NewPagerduty(cfg *config.Config) *Pagerduty {
	return &Pagerduty{
		Config:     cfg,
		identifier: "slack",
	}
}

func (p *Pagerduty) Send(msg *Message, alerterConfig *AlerterConfig) error {
	log.Debugf("%v: Sending message %v", p.identifier, msg.UUID)
	return nil
}

func (p *Pagerduty) Identifier() string {
	return p.identifier
}
