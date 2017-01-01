package event

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	BUFFER_LEN   = 1000
	WORKER_COUNT = 20
	WORKER_SLEEP = time.Duration(500) * time.Millisecond
)

type Queue struct {
	Identifier string
	channel    chan *Event
}

type Client struct {
	queue *Queue
}

type Event struct {
	Type    string
	Message string
}

func NewQueue() *Queue {
	return &Queue{
		Identifier: "event",
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
	select {
	case c.queue.channel <- &Event{
		Type:    key,
		Message: value,
	}:
		return nil
	default:
		return fmt.Errorf("Event queue is full; discarding event (Type: %v Message: %v)",
			key, value)
	}
}
