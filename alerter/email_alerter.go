package alerter

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/config"
)

const (
	DEFAULT_EMAIL_FROM = "9volt-alerter@example.com"
	DEFAULT_EMAIL_AUTH = "plain"
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

	from := DEFAULT_EMAIL_FROM

	if _, ok := alerterConfig.Options["from"]; ok {
		from = alerterConfig.Options["from"]
	}

	auth, err := e.generateAuth(alerterConfig)
	if err != nil {
		return fmt.Errorf("Unable to generate auth instance for sending email: %v", err.Error())
	}

	emailMsg := e.generateMessage(msg, alerterConfig)

	if err := smtp.SendMail(alerterConfig.Options["address"], auth, from, []string{alerterConfig.Options["to"]}, emailMsg); err != nil {
		return fmt.Errorf("Unable to send email alert to %v via %v (from: %v): %v",
			alerterConfig.Options["to"], alerterConfig.Options["address"], from, err.Error())
	}

	return nil
}

func (e *Email) generateMessage(msg *Message, alerterConfig *AlerterConfig) []byte {
	emailMessage := fmt.Sprintf("To: %v\r\n"+
		"Subject: %v\r\n"+
		"\r\n"+
		"Detailed Message: %v\r\n"+
		"Error Details: %v\r\n", alerterConfig.Options["to"], msg.Title, msg.Text, msg.Contents["ErrorDetails"],
	)

	return []byte(emailMessage)
}

func (e *Email) generateAuth(alerterConfig *AlerterConfig) (smtp.Auth, error) {
	requiredFields := []string{"username", "password"}

	// If username or password is not set, return nil for auth
	for _, v := range requiredFields {
		if _, ok := alerterConfig.Options[v]; !ok {
			return nil, nil
		}
	}

	authType := DEFAULT_EMAIL_AUTH

	// If auth is set, verify that it is either 'md5' or 'plain'
	if _, ok := alerterConfig.Options["auth"]; ok {
		if alerterConfig.Options["auth"] != "plain" && alerterConfig.Options["auth"] != "md5" {
			return nil, fmt.Errorf("'auth' must be set to either 'plain' or 'md5'")
		} else {
			authType = alerterConfig.Options["auth"]
		}
	}

	// This *should* be already taken care of by ValidateConfig(), but JIC
	if _, ok := alerterConfig.Options["address"]; !ok {
		return nil, errors.New("'address' should exist in alerter config options")
	}

	// PlainAuth requires the address to be passed without a port
	host := strings.Split(alerterConfig.Options["address"], ":")

	switch authType {
	case "plain":
		return smtp.PlainAuth("", alerterConfig.Options["username"], alerterConfig.Options["password"], host[0]), nil
	case "md5":
		return smtp.CRAMMD5Auth(alerterConfig.Options["username"], alerterConfig.Options["password"]), nil
	default:
		return nil, fmt.Errorf("Unhandled authType '%v': shouldn't hit this case, bug?", authType)
	}

}

func (e *Email) Identify() string {
	return e.Identifier
}

func (e *Email) ValidateConfig(alerterConfig *AlerterConfig) error {
	if len(alerterConfig.Options) == 0 {
		return errors.New("Options must be filled out")
	}

	requiredFields := []string{"to", "address"}
	errorList := make([]string, 0)

	for _, v := range requiredFields {
		if _, ok := alerterConfig.Options[v]; !ok {
			errorList = append(errorList, fmt.Sprintf("'%v' must be present in options", v))
		}
	}

	// Ensure that 'auth' is correct (if set)
	if _, ok := alerterConfig.Options["auth"]; ok {
		if alerterConfig.Options["auth"] != "plain" && alerterConfig.Options["auth"] != "md5" {
			errorList = append(errorList, "'auth' must be either 'plain' or 'md5'")
		}
	}

	// if username is present, a password must be set as well
	if _, ok := alerterConfig.Options["username"]; ok {
		if _, found := alerterConfig.Options["password"]; !found {
			errorList = append(errorList, "'password' must be present if 'username' is set")
		}
	}

	if len(errorList) != 0 {
		fullError := strings.Join(errorList, "; ")
		return errors.New(fullError)
	}

	return nil
}
