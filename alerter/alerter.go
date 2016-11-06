package alerter

import (
	"encoding/json"
	"errors"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/relistan/go-director"
	"github.com/satori/go.uuid"

	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/util"
)

type IAlerter interface {
	Send(*Message, *AlerterConfig) error
	Identifier() string
}

type AlerterConfig struct {
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Options     map[string]string `json:"options"`
}

type Alerter struct {
	Identifier     string
	MemberID       string
	Config         *config.Config
	Alerters       map[string]IAlerter
	MessageChannel <-chan *Message
	Looper         director.Looper
}

type Message struct {
	Key      string
	Contents map[string]string
	Source   string
	Resolve  bool
	UUID     string
}

func New(cfg *config.Config, messageChannel <-chan *Message) *Alerter {
	return &Alerter{
		Identifier:     "alerter",
		MemberID:       util.GetMemberID(cfg.ListenAddress),
		Config:         cfg,
		MessageChannel: messageChannel,
		Looper:         director.NewFreeLooper(director.FOREVER, make(chan error)),
	}
}

func (a *Alerter) Start() error {
	log.Infof("%v: Starting alerter components...", a.Identifier)

	// Instantiate our alerters
	pd := NewPagerduty(a.Config)
	sl := NewSlack(a.Config)

	a.Alerters = map[string]IAlerter{
		pd.Identifier(): pd,
		sl.Identifier(): sl,
	}

	// Launch our alerter message handler
	go a.run()

	return nil
}

func (a *Alerter) run() error {
	a.Looper.Loop(func() error {
		msg := <-a.MessageChannel

		// tag message
		msg.UUID = uuid.NewV4().String()

		log.Debugf("%v: Received message (%v) from checker '%v' -> '%v'", msg.UUID, a.Identifier, msg.Source, msg.Key)

		go a.handleMessage(msg)

		return nil
	})

	return nil
}

func (a *Alerter) handleMessage(msg *Message) error {
	// validate message contents
	if err := a.validateMessage(msg); err != nil {
		log.Errorf("%v: Unable to validate message (%v): %v", a.Identifier, msg.UUID, err.Error())
		return err
	}

	// fetch alert configuration
	jsonAlerterConfig, err := a.Config.DalClient.FetchAlerterConfig(msg.Key)
	if err != nil {
		log.Errorf("%v: Unable to fetch alerter config for message %v: %v", a.Identifier, msg.UUID, err.Error())
		return err
	}

	// try to unmarshal the data
	var alerterConfig *AlerterConfig

	if err := json.Unmarshal([]byte(jsonAlerterConfig), alerterConfig); err != nil {
		log.Errorf("%v: Unable to unmarshal alerter config for message %v: %v", a.Identifier, msg.UUID, err.Error())
		return err
	}

	// check if we have given alerter
	if _, ok := a.Alerters[alerterConfig.Type]; !ok {
		err := fmt.Errorf("%v: Unable to find any alerter named %v", a.Identifier, alerterConfig.Type)
		log.Error(err.Error())
		return err
	}

	// send the actual alert
	log.Debugf("%v: Sending %v to alerter %v!", a.Identifier, msg.UUID, alerterConfig.Type)
	if err := a.Alerters[alerterConfig.Type].Send(msg, alerterConfig); err != nil {
		log.Errorf("%v: Unable to complete message send for %v: %v", a.Identifier, msg.UUID, err.Error())
		return err
	}

	log.Debugf("%v: Successfully sent %v alert message %v", a.Identifier, alerterConfig.Type, msg.UUID)
	return nil
}

// Perform message validation; return err if one of required fields is not filled out
func (a *Alerter) validateMessage(msg *Message) error {
	if msg.Key == "" {
		return errors.New("Message must have the 'Key' value filled out")
	}

	if msg.Source == "" {
		return errors.New("Message must have the 'Source' value filled out")
	}

	if msg.Contents == nil {
		return errors.New("Message 'Contents' must be filled out")
	}

	return nil
}
