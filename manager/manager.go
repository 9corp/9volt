// TODO: If our 'member dir' disappears, should we stop all monitors?
// TODO: Does a state change mean we should cease monitoring?

package manager

import (
	"context"
	"fmt"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"github.com/relistan/go-director"

	"github.com/9corp/9volt/alerter"
	"github.com/9corp/9volt/base"
	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/monitor"
	"github.com/9corp/9volt/overwatch"
	"github.com/9corp/9volt/state"
)

type Manager struct {
	MemberID      string
	Config        *config.Config
	Looper        director.Looper
	Monitor       *monitor.Monitor
	OverwatchChan chan<- *overwatch.Message

	base.Component
}

func New(cfg *config.Config, messageChannel chan *alerter.Message, stateChannel chan *state.Message, overwatchChan chan<- *overwatch.Message) (*Manager, error) {
	return &Manager{
		MemberID:      cfg.MemberID,
		Config:        cfg,
		Looper:        director.NewFreeLooper(director.FOREVER, make(chan error)),
		Monitor:       monitor.New(cfg, messageChannel, stateChannel),
		OverwatchChan: overwatchChan,
		Component: base.Component{
			Identifier: "manager",
		},
	}, nil
}

func (m *Manager) Start() error {
	log.Infof("%v: Starting manager components...", m.Identifier)

	m.Component.Ctx, m.Component.Cancel = context.WithCancel(context.Background())

	go m.run()

	return nil
}

func (m *Manager) Stop() error {
	if m.Component.Cancel == nil {
		log.Warningf("%v: Looks like .Cancel is nil; is this expected?", m.Identifier)
	} else {
		m.Component.Cancel()
	}

	m.Monitor.StopAll()

	return nil
}

func (m *Manager) run() error {
	memberConfigDir := fmt.Sprintf("cluster/members/%v/config", m.MemberID)

	watcher := m.Config.DalClient.NewWatcher(memberConfigDir, true)

	for {
		resp, err := watcher.Next(m.Component.Ctx)
		if err != nil {
			if err.Error() == "context canceled" {
				log.Debugf("%v: Received a notice to shutdown", m.Identifier)
				break
			}

			m.Config.EQClient.AddWithErrorLog("error",
				fmt.Sprintf("%v: Unexpected watcher error: %v", m.Identifier, err.Error()))

			// Tell overwatch that something bad just happened
			m.OverwatchChan <- &overwatch.Message{
				Error:     fmt.Errorf("Unexpected watcher error: %v", err),
				Source:    fmt.Sprintf("%v.run", m.Identifier),
				ErrorType: overwatch.ETCD_WATCHER_ERROR,
			}

			// Let overwatch determine whether to shut us down
			continue
		}

		if m.ignorableWatcherEvent(resp) {
			log.Debugf("%v: Received an ignorable watcher '%v' event for key '%v'",
				m.Identifier, resp.Action, resp.Node.Key)
			continue
		}

		log.Debugf("%v: Received a '%v' watcher event for '%v' (value: '%v')",
			m.Identifier, resp.Action, resp.Node.Key, resp.Node.Value)

		switch resp.Action {
		case "set":
			go m.Monitor.Handle(monitor.START, path.Base(resp.Node.Key), resp.Node.Value)
		case "delete":
			go m.Monitor.Handle(monitor.STOP, path.Base(resp.Node.Key), resp.Node.Value)
		default:
			m.Config.EQClient.AddWithErrorLog("error",
				fmt.Sprintf("%v: Received an unrecognized action '%v' - skipping", m.Identifier, resp.Action))
		}
	}

	log.Debugf("%v: Exiting", m.Identifier)

	return nil
}

// Determine if a specific event can be ignored
func (m *Manager) ignorableWatcherEvent(resp *client.Response) bool {
	if resp == nil {
		log.Debugf("%v: Received a nil etcd response - bug?", m.Identifier)
		return true
	}

	// Ignore anything that is `config` related
	if path.Base(resp.Node.Key) == "config" {
		return true
	}

	return false
}
