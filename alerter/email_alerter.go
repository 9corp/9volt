package alerter

import (
	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/config"
)

type Email struct {
	Config     *config.Config
	Identifier string
}

func NewEmail(cfg *config.Config) *Email {
	return &Email{
		Config:     cfg,
		Identifier: "email",
	}
}

func (e *Email) Send(msg *Message, alerterConfig *AlerterConfig) error {
	log.Debugf("%v: Sending message %v...", e.Identifier, msg.uuid)
	return nil
}

func (e *Email) Identify() string {
	return e.Identifier
}

func (e *Email) ValidateConfig(alerterConfig *AlerterConfig) error {
	return nil
}
