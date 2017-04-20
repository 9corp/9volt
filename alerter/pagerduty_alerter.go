package alerter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/inconshreveable/log15"

	"github.com/9corp/9volt/config"
)

const (
	EVENT_TYPE_TRIGGER = "trigger"
	EVENT_TYPE_RESOLVE = "resolve"
)

type Pagerduty struct {
	Config     *config.Config
	Identifier string
	Log        log15.Logger
}

func NewPagerduty(cfg *config.Config, logger log15.Logger) *Pagerduty {
	return &Pagerduty{
		Config:     cfg,
		Identifier: "pagerduty",
		Log:        logger.New("type", "pagerduty"),
	}
}

func (p *Pagerduty) Send(msg *Message, alerterConfig *AlerterConfig) error {
	p.Log.Debug("Sending message", "uuid", msg.uuid)

	// generate event
	event := p.generateEvent(msg, alerterConfig)

	// send event
	resp, err := pagerduty.CreateEvent(*event)
	if err != nil {
		return fmt.Errorf("Unable to create pagerduty event for %v: %v", msg.Source, err.Error())
	}

	p.Log.Debug("Pagerduty response", "status", resp.Status, "msg", resp.Message, "incidentKey", resp.IncidentKey)

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
