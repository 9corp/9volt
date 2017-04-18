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
	AddWithErrorLog(string, string) error
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
		log.Debugf("%v: Already running - nothing to do", q.Identifier)
		return nil
	}

	q.Running = true

	// launch worker pool
	log.Debugf("%v: Launching %v queue workers", q.Identifier, WORKER_COUNT)

	for i := 1; i <= WORKER_COUNT; i++ {
		go q.runWorker(i)
	}

	return nil
}

func (q *Queue) Stop() error {
	log.Debugf("%v: Workers are paused", q.Identifier)

	q.Pause = true

	return nil
}

func (q *Queue) runWorker(id int) {
	log.Debugf("%v: Event worker #%v started", q.Identifier, id)

	for {
		e := <-q.channel

		if q.Pause {
			log.Debugf("%v: In pause state; dropping inbound message(s)", q.Identifier)
			continue
		}

		log.Debugf("%v: Worker #%v received new event! Type: %v Message: %v", q.Identifier,
			id, e.Type, e.Message)

		// marshal the event
		eventBlob, err := json.Marshal(e)
		if err != nil {
			log.Errorf("%v: Unable to marshal event '%v' to JSON (worker #%v): %v",
				q.Identifier, e, id, err)
			continue
		}

		// write it to etcd
		fullKey := fmt.Sprintf("event/%v-%v", e.Type, util.RandomString(6, true))

		if err := q.DalClient.Set(fullKey, string(eventBlob), &dal.SetOptions{Dir: false, TTLSec: MAX_EVENT_AGE, PrevExist: ""}); err != nil {
			log.Errorf("%v: Unable to save event blob '%v' to path '%v' to etcd (worker: #%v): %v",
				q.Identifier, e, fullKey, id, err)
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

func (c *Client) AddWithErrorLog(key, value string) error {
	log.Error(value)

	if err := c.Add(key, value); err != nil {
		return err
	}

	return nil
}
