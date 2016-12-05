package alerter

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/config"
)

const (
	RECOVERED_COLOR = "#36a64f" // green
	WARNING_COLOR   = "#ff9400" // orange
	CRITICAL_COLOR  = "#ff0000" // red
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

// Ensure that our alerter config contains all of the necessary information
func (s *Slack) ValidateConfig(alerterConfig *AlerterConfig) error {
	if len(alerterConfig.Options) == 0 {
		return errors.New("Options must be filled out")
	}

	requiredFields := []string{"token", "channel"}
	errorList := make([]string, 0)

	for _, v := range requiredFields {
		if _, ok := alerterConfig.Options[v]; !ok {
			errorList = append(errorList, fmt.Sprintf("'%v' must be present in options", v))
		}
	}

	if len(errorList) != 0 {
		fullError := strings.Join(errorList, "; ")
		return errors.New(fullError)
	}

	return nil
}
