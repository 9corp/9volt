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
	RESOLVE_COLOR  = "#36a64f" // green
	WARNING_COLOR  = "#ff9400" // orange
	CRITICAL_COLOR = "#ff0000" // red

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
	messageColor := RESOLVE_COLOR
	messageIconURL := ""

	// If present, use custom username
	if _, ok := alerterConfig.Options["username"]; ok {
		messageUsername = alerterConfig.Options["username"]
	}

	// If present, use custom icon url
	if _, ok := alerterConfig.Options["icon-url"]; ok {
		messageIconURL = alerterConfig.Options["icon-url"]
	}

	if msg.Type == "critical" {
		messageColor = CRITICAL_COLOR
	} else if msg.Type == "warning" {
		messageColor = WARNING_COLOR
	}

	attachment := slack.Attachment{
		Color:    messageColor,
		Fallback: msg.Title,
		Title:    msg.Title,
		Text:     msg.Text,
	}

	// if not a recovery, attach error details
	attachment.Fields = []slack.AttachmentField{
		{
			Title: "Error Details",
			Value: msg.Contents["ErrorDetails"],
		},
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
