package alerter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PagerDuty/go-pagerduty"
	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/config"
)

const (
	EVENT_TYPE_TRIGGER = "trigger"
	EVENT_TYPE_RESOLVE = "resolve"
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

	// generate event
	event := p.generateEvent(msg, alerterConfig)

	// send event
	resp, err := pagerduty.CreateEvent(*event)
	if err != nil {
		return fmt.Errorf("Unable to create pagerduty event for %v: %v", msg.Source, err.Error())
	}

	log.Debugf("Response Status: %v Message: %v IncidentKey: %v", resp.Status, resp.Message, resp.IncidentKey)

	return nil
}

func (p *Pagerduty) generateEvent(msg *Message, alertConfig *AlerterConfig) *pagerduty.Event {
	eventType := EVENT_TYPE_RESOLVE

	if msg.Type != "resolve" {
		eventType = EVENT_TYPE_TRIGGER
	}

	event := &pagerduty.Event{
		ServiceKey:  alertConfig.Options["token"],
		Type:        eventType,
		IncidentKey: msg.Source,
		Description: msg.Title,
		Client:      "9volt",
		Details: map[string]string{
			"error details":    msg.Contents["ErrorDetails"],
			"detailed message": msg.Text,
			"description":      msg.Description,
		},
		// ClientURL:   "https://url-to-9volt-ui?",
	}

	return event
}

func (p *Pagerduty) Identify() string {
	return p.Identifier
}

func (p *Pagerduty) ValidateConfig(alerterConfig *AlerterConfig) error {
	if len(alerterConfig.Options) == 0 {
		return errors.New("Options must be filled out")
	}

	requiredFields := []string{"token"}
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
