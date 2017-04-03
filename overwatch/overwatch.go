// Overwatch package is responsible for:
//
//   - listening for bad events from components
//   - gracefully stopping all affected components
//   - watching and waiting for the dependency to recover
//   - starting all components back up
//
package overwatch

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/relistan/go-director"

	"github.com/9corp/9volt/base"
	"github.com/9corp/9volt/config"
)

const (
	WATCH_RETRY_INTERVAL  = time.Duration(5) * time.Second
	HEALTH_WATCH_DURATION = time.Duration(30) * time.Second

	ETCD_WATCHER_ERROR int = iota
)

type Overwatch struct {
	Config       *config.Config
	WatchChannel <-chan *Message
	Looper       director.Looper
	WatchLooper  director.Looper
	Components   []base.IComponent
	activeWatch  bool

	base.Component
}

type Message struct {
	Source    string
	Error     error
	ErrorType int
}

func New(cfg *config.Config, watchChannel <-chan *Message, components []base.IComponent) *Overwatch {
	return &Overwatch{
		Config:       cfg,
		WatchChannel: watchChannel,
		Components:   components,
		Looper:       director.NewFreeLooper(director.FOREVER, make(chan error)),
		WatchLooper:  director.NewTimedLooper(director.FOREVER, WATCH_RETRY_INTERVAL, make(chan error)),
		Component: base.Component{
			Identifier: "overwatch",
		},
	}
}

func (o *Overwatch) Start() error {
	log.Debugf("%v: Launching watcher component...", o.Identifier)

	go o.runListener()

	return nil
}

// TODO
func (o *Overwatch) Stop() error {
	return nil
}

func (o *Overwatch) runListener() error {
	// Listen for bad events from components
	o.Looper.Loop(func() error {
		msg := <-o.WatchChannel

		log.Warningf("%v: Received overwatch event from %v (error: %v)", o.Identifier, msg.Source, msg.Error)

		if o.activeWatch {
			log.Debugf("%v: Watcher already activated, nothing else left to do", o.Identifier)
			return nil
		}

		o.activeWatch = true

		// Okay, let's stop everything!
		go o.stopTheWorld(msg)

		return nil
	})

	return nil
}

// Wrapper for: stopping all components, starting dep watch and re-starting all
// components upon dependency recovery
func (o *Overwatch) stopTheWorld(msg *Message) error {
	// Stop all components
	for _, v := range o.Components {
		log.Debugf("%v: Stopping component '%v'...", o.Identifier, v.Identify())

		if err := v.Stop(); err != nil {
			// TODO: Do something smarter here
			log.Errorf("%v: Unable to stop component '%v': %v", o.Identifier, v.Identify(), err)
			continue
		}
	}

	// Start watching dependency
	go o.handleWatch(msg)

	return nil
}

// Launch appropriate dependency watcher
func (o *Overwatch) handleWatch(msg *Message) error {
	switch msg.ErrorType {
	case ETCD_WATCHER_ERROR:
		go o.beginEtcdWatch()
	default:
		log.Errorf("%v: Unknown error type '%v' - unable to complete handleWatch(); (error: %v)", o.Identifier, msg.ErrorType, msg.Error)
		return fmt.Errorf("%v: Unknown error type '%v' - unable to complete handleWatch(); (error: %v)", o.Identifier, msg.ErrorType, msg.Error)
	}

	return nil
}

func (o *Overwatch) beginEtcdWatch() error {
	// watch etcd for $time, start components back up if no errors present
	tmpWatchChannel := make(chan bool, 1)

	go func() {
		startTime := time.Now()

	MainLoop:
		for {
			select {
			case <-tmpWatchChannel:
				// errors occurred, reset timer
				log.Warning("An error occurred in the watcher; continue watching")
				startTime = time.Now()
			default:
				if time.Now().Sub(startTime) >= HEALTH_WATCH_DURATION {
					log.Warningf("Watcher has not reported errors for %v; starting everything back up", HEALTH_WATCH_DURATION)

					o.activeWatch = false
					o.WatchLooper.Quit()

					if err := o.startTheWorld(); err != nil {
						// TODO: What do we do here?
						log.Errorf("Unable to start the world after recovery: %v", err)
					}

					break MainLoop
				}

				// either timer has not elapsed yet, or we've received errors
			}

			time.Sleep(time.Duration(1) * time.Second)
		}

		log.Warningf("%v: Error watch goroutine exiting", o.Identifier)
	}()

	// Start the actual watcher
	go func() {
		o.WatchLooper.Loop(func() error {
			watcher, err := o.Config.DalClient.NewWatcherForOverwatch("/", true)
			if err != nil {
				log.Errorf("Unable to begin watching '/'; retrying in %v: %v", WATCH_RETRY_INTERVAL, err)
				return nil
			}

			for {
				_, err := watcher.Next(context.Background())
				if err != nil {
					log.Errorf("Experienced error during watch, recreating watcher: %v", err)
					tmpWatchChannel <- true
					break
				}
			}

			return nil
		})

		log.Warningf("%v: Watcher goroutine exiting...", o.Identifier)
	}()

	return nil
}

func (o *Overwatch) startTheWorld() error {
	log.Warning("OKAY, STARTING THE WORLD BACK UP!")
	return nil
}
