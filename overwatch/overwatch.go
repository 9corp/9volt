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
	// "errors"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/relistan/go-director"

	"github.com/9corp/9volt/base"
	"github.com/9corp/9volt/config"
)

const (
	WATCH_RETRY_INTERVAL  = time.Duration(5) * time.Second
	HEALTH_WATCH_DURATION = time.Duration(10) * time.Second

	ETCD_WATCHER_ERROR int = iota
	ETCD_GENERIC_ERROR
)

type Overwatch struct {
	Config       *config.Config
	WatchChannel <-chan *Message
	Looper       director.Looper
	WatchLooper  director.Looper
	Components   []base.IComponent
	activeWatch  bool
	Log          log.FieldLogger

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
		Log:          log.WithField("pkg", "overwatch"),
		Components:   components,
		Looper:       director.NewFreeLooper(director.FOREVER, make(chan error)),
		WatchLooper:  director.NewTimedLooper(director.FOREVER, WATCH_RETRY_INTERVAL, make(chan error)),
		Component: base.Component{
			Identifier: "overwatch",
		},
	}
}

func (o *Overwatch) Start() error {

	o.Log.Debug("Launching watcher component...")

	go o.runListener()

	return nil
}

// TODO
func (o *Overwatch) Stop() error {
	return nil
}

func (o *Overwatch) runListener() error {
	llog := o.Log.WithField("method", "runListener")

	// Listen for bad events from components
	o.Looper.Loop(func() error {
		msg := <-o.WatchChannel

		llog.WithFields(log.Fields{
			"source": msg.Source,
			"err":    msg.Error,
		}).Warning("Received overwatch event")

		if o.activeWatch {
			llog.Debug("Watcher already activated, nothing else left to do")
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
	o.Log.Warning("Stopping all components", o.Identifier)

	// Stop all components
	for _, v := range o.Components {
		o.Log.WithField("component", v.Identify()).Warning("Stopping component...")

		if err := v.Stop(); err != nil {
			// TODO: Do something smarter here
			o.Log.WithFields(log.Fields{
				"component": v.Identify(),
				"err":       err,
			}).Error("Unable to stop component")
			continue
		}
	}

	// Start watching dependency
	go o.handleWatch(msg)

	// And tell the healthcheck we're not doing great
	o.Config.Health.Write(false, fmt.Sprintf("%v: %v", msg.Source, msg.Error))

	return nil
}

// Launch appropriate dependency watcher
func (o *Overwatch) handleWatch(msg *Message) error {
	switch msg.ErrorType {
	case ETCD_WATCHER_ERROR:
		go o.beginEtcdWatch()
	case ETCD_GENERIC_ERROR:
		go o.beginEtcdWatch()
	default:
		o.Log.WithFields(log.Fields{
			"errorType": msg.ErrorType,
			"err":       msg.Error,
		}).Error("Unknown error type; unable to complete handleWatch()")

		return fmt.Errorf("%v: Unknown error type '%v' - unable to complete handleWatch(); (error: %v)", o.Identifier, msg.ErrorType, msg.Error)
	}

	return nil
}

func (o *Overwatch) beginEtcdWatch() error {
	// watch etcd for $time, start components back up if no errors present
	tmpWatchChannel := make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())

	go func(cancel context.CancelFunc) {
		startTime := time.Now()

	OUTER:
		for {
			select {
			case <-tmpWatchChannel:
				// errors occurred, reset timer
				o.Log.Debug("Errors with backend, continuing test watch")
				startTime = time.Now()
			default:
				if time.Now().Sub(startTime) >= HEALTH_WATCH_DURATION {
					o.Log.WithField("watchPeriod", HEALTH_WATCH_DURATION).Warning(
						"Watcher has not reported errors during watchPeriod; starting everything back up")

					o.activeWatch = false
					cancel()

					if err := o.startTheWorld(); err != nil {
						// TODO: Starting the components failed, what now?
						o.Log.WithField("err", err).Error("Unable to start the world after recovery")
					}

					break OUTER
				}

				// either timer has not elapsed yet, or we've received errors
			}

			time.Sleep(time.Duration(1) * time.Second)
		}

		o.Log.Debug("Primary watcher goroutine exiting")
	}(cancel)

	// Start the actual watcher
	go func(ctx context.Context) {
	OUTER:
		for {
			watcher, err := o.Config.DalClient.NewWatcherForOverwatch("/", true)
			if err != nil {
				o.Log.WithFields(log.Fields{
					"retryInterval": WATCH_RETRY_INTERVAL,
					"err":           err,
				}).Error("Unable to begin watching '/'; retrying...")
				time.Sleep(WATCH_RETRY_INTERVAL)
				continue
			}

			for {
				_, err := watcher.Next(ctx)
				if err != nil {
					if err.Error() == "context canceled" {
						o.Log.Debug("Etcd watcher has been cancelled")
						break OUTER
					} else {
						// log.Debugf("%v: Experienced error during watch, recreating watcher: %v", o.Identifier, err)
						tmpWatchChannel <- true
						break
					}
				}
			}
		}

		o.Log.Debug("Etcd watcher goroutine exiting...")
	}(ctx)

	return nil
}

func (o *Overwatch) startTheWorld() error {
	errorList := make([]string, 0)

	for _, v := range o.Components {
		log.Debugf("%v: Starting up '%v' component", o.Identifier, v.Identify())
		if err := v.Start(); err != nil {
			errorList = append(errorList, err.Error())

			o.Log.WithFields(log.Fields{
				"component": v.Identify(),
				"err":       err,
			}).Error("Unable to start component")
		}
	}

	if len(errorList) != 0 {
		return fmt.Errorf("Ran into one or more errors during component startup: %v", strings.Join(errorList, "; "))
	}

	// Finally, update the healthcheck to say we've recovered
	o.Config.Health.Write(true, fmt.Sprintf("Recovered from previous error: '%v'", o.Config.Health.Message))

	return nil
}
