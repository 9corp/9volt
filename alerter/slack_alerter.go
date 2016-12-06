package alerter

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"

	"github.com/9corp/9volt/config"
)

const (
	RECOVERED_COLOR = "#36a64f" // green
	WARNING_COLOR   = "#ff9400" // orange
	CRITICAL_COLOR  = "#ff0000" // red

	DEFAULT_SLACK_USERNAME = "9volt-bot"
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

	// generate slack message params
	params := s.generateParams(msg, alerterConfig)

	// create a new slack client
	client := slack.New(alerterConfig.Options["token"])

	// do not set message text - message text is handled as an attachment inside params
	_, _, err := client.PostMessage(alerterConfig.Options["channel"], "", *params)
	if err != nil {
		return err
	}

	return nil
}

// Generate slack (post) message parameters (configure what the message looks like, etc.)
func (s *Slack) generateParams(msg *Message, alerterConfig *AlerterConfig) *slack.PostMessageParameters {
	messageUsername := DEFAULT_SLACK_USERNAME
	messageColor := RECOVERED_COLOR
	messageIconURL := ""
	messageHeader := "Recovered"

	// If present, use custom username
	if _, ok := alerterConfig.Options["username"]; ok {
		messageUsername = alerterConfig.Options["username"]
	}

	// If present, use custom icon url
	if _, ok := alerterConfig.Options["iconURL"]; ok {
		messageIconURL = alerterConfig.Options["iconURL"]
	}

	if msg.Critical {
		messageColor = CRITICAL_COLOR
		messageHeader = "Critical"
	} else if msg.Warning {
		messageColor = WARNING_COLOR
		messageHeader = "Warning"
	}

	attachment := slack.Attachment{
		Color:    messageColor,
		Fallback: fmt.Sprintf("%v: %v", strings.ToUpper(messageHeader), msg.Source),
		Title:    fmt.Sprintf("%v: %v", strings.ToUpper(messageHeader), msg.Source),
		Text:     msg.Text,
		Fields: []slack.AttachmentField{
			slack.AttachmentField{
				Title: "Thresholds",
				Value: fmt.Sprintf("Warning: %v Critical: %v", msg.Contents["WarningThreshold"], msg.Contents["CriticalThreshold"]),
			},
		},
	}

	// Prepend additional "attempts" attachment if this is a recovery
	if msg.Resolve {
		attemptAttachment := slack.AttachmentField{
			Title: "Attempts",
			Value: fmt.Sprint(msg.Count),
		}

		attachment.Fields = append([]slack.AttachmentField{attemptAttachment}, attachment.Fields...)
	}

	params := slack.PostMessageParameters{
		Username:    messageUsername,
		IconURL:     messageIconURL,
		Attachments: []slack.Attachment{attachment},
	}

	return &params
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
