package event

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

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
	Identifier string
	DalClient  dal.IDal
	MemberID   string
	channel    chan *Event
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
		Identifier: "event",
		DalClient:  dalClient,
		MemberID:   memberID,
		channel:    make(chan *Event, BUFFER_LEN),
	}
}

func (q *Queue) Start() error {
	// launch worker pool
	log.Debugf("%v: Launching %v queue workers", q.Identifier, WORKER_COUNT)

	for i := 1; i <= WORKER_COUNT; i++ {
		go q.runWorker(i)
	}

	return nil
}

func (q *Queue) runWorker(id int) {
	log.Debugf("%v: Event worker #%v started", q.Identifier, id)

	for {
		e := <-q.channel

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
		fullKey := fmt.Sprintf("event/%v-%v", e.Type, util.RandomString(6, false))

		if err := q.DalClient.Set(fullKey, string(eventBlob), false, MAX_EVENT_AGE, ""); err != nil {
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
