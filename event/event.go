// Event package is responsible for receiving events from 9volt components and
// dumping them to etcd. The event queue is powered by WORKER_COUNT workers and
// has a BUFFER_LEN buffer.
//
// Unlike other components, the event queue does NOT get shutdown in the event
// of a backend failure - it gets *paused*. In the *pause* state, the event queue
// will simply discard any inbound messages and will NOT attempt to save them to
// the backend.
//
// This is done in order to avoid potential race conditions where lagging/slower
// components continue to write to the event queue even though they've been asked
// to shutdown.
package event

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/base"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/util"
)

const (
	BUFFER_LEN    = 1000
	WORKER_COUNT  = 20
	WORKER_SLEEP  = time.Duration(500) * time.Millisecond
	MAX_EVENT_AGE = 86400 // 24 hours
)

type Queue struct {
	Log       log.FieldLogger
	DalClient dal.IDal
	MemberID  string
	channel   chan *Event
	Running   bool // do not allow more than one Start() to be issued; resets Pause
	Pause     bool // Pause set via Stop(); causes messages to be discarded

	base.Component
}

//go:generate counterfeiter -o ../fakes/eventfakes/fake_client.go event.go IClient

type IClient interface {
	Add(string, string) error
	AddWithErrorLog(string, log.FieldLogger, log.Fields) error
	AddWithLog(string, string, log.FieldLogger, log.Fields) error
}

type Client struct {
	queue *Queue
}

type Event struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	MemberId  string    `json:"memberid"`
}

func NewQueue(memberID string, dalClient dal.IDal) *Queue {
	return &Queue{
		Log:       log.WithFields(log.Fields{"pkg": "event"}),
		DalClient: dalClient,
		MemberID:  memberID,
		channel:   make(chan *Event, BUFFER_LEN),
		Component: base.Component{
			Identifier: "event",
		},
	}
}

func (q *Queue) Start() error {
	q.Pause = false

	if q.Running {
		q.Log.Debug("Already running - nothing to do")
		return nil
	}

	q.Running = true

	// launch worker pool
	q.Log.Debugf("Launching %v queue workers", WORKER_COUNT)

	for i := 1; i <= WORKER_COUNT; i++ {
		go q.runWorker(i)
	}

	return nil
}

func (q *Queue) Stop() error {
	q.Log.Debug("Workers are paused")

	q.Pause = true

	return nil
}

func (q *Queue) runWorker(id int) {
	llog := q.Log.WithFields(log.Fields{"id": id})

	llog.Debug("Event worker started")

	for {
		e := <-q.channel

		if q.Pause {
			llog.Debug("In pause state; dropping inbound message(s)")
			continue
		}

		llog.WithFields(log.Fields{"type": e.Type, "msg": e.Message}).Debug("Worker received new event")

		// marshal the event
		eventBlob, err := json.Marshal(e)
		if err != nil {
			llog.WithFields(log.Fields{"event": e, "err": err}).Errorf("Unable to marshal event")
			continue
		}

		// write it to etcd
		fullKey := fmt.Sprintf("event/%v-%v", e.Type, util.RandomString(6, true))

		if err := q.DalClient.Set(fullKey, string(eventBlob), &dal.SetOptions{Dir: false, TTLSec: MAX_EVENT_AGE, PrevExist: ""}); err != nil {
			llog.WithFields(log.Fields{"event": e, "path": fullKey, "err": err}).Error("Unable to save event blob")
		}

		// Artificially slow down queue workers (and prevent an etcd write flood)
		time.Sleep(WORKER_SLEEP)
	}
}

func (q *Queue) NewClient() *Client {
	return &Client{
		queue: q,
	}
}

// Attempt to send an event to the event channel; discard event if event channel is full
func (c *Client) Add(key, value string) error {
	key = strings.ToLower(key)

	select {
	case c.queue.channel <- &Event{
		Type:      key,
		Message:   value,
		MemberId:  c.queue.MemberID,
		Timestamp: time.Now(),
	}:
		return nil
	default:
		return fmt.Errorf("Event queue is full; discarding event (Type: %v Message: %v)",
			key, value)
	}
}

// Helper for AddWithLog()
func (c *Client) AddWithErrorLog(value string, logger log.FieldLogger, fields log.Fields) error {
	return c.AddWithLog("error", value, logger, fields)
}

func (c *Client) AddWithLog(key, value string, logger log.FieldLogger, fields log.Fields) error {
	switch key {
	case "error":
		logger.WithFields(fields).Error(value)
	case "warning":
		logger.WithFields(fields).Warning(value)
	default:
		logger.WithFields(fields).Info(value)
	}

	fieldEntries := make([]string, 0)

	for k, v := range fields {
		fieldEntries = append(fieldEntries, fmt.Sprintf("%v=%v", k, v))
	}

	eventMessage := value + " [" + strings.Join(fieldEntries, " ") + "]"

	if err := c.Add(key, eventMessage); err != nil {
		return err
	}

	return nil
}
