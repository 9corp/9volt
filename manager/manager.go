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
	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/monitor"
	"github.com/9corp/9volt/state"
)

type Manager struct {
	MemberID   string
	Identifier string
	Config     *config.Config
	Looper     director.Looper
	Monitor    *monitor.Monitor
}

func New(cfg *config.Config, messageChannel chan *alerter.Message, stateChannel chan *state.Message) (*Manager, error) {
	return &Manager{
		Identifier: "manager",
		MemberID:   cfg.MemberID,
		Config:     cfg,
		Looper:     director.NewFreeLooper(director.FOREVER, make(chan error)),
		Monitor:    monitor.New(cfg, messageChannel, stateChannel),
	}, nil
}

func (m *Manager) Start() error {
	log.Infof("%v: Starting manager components...", m.Identifier)

	go m.run()

	return nil
}

func (m *Manager) run() error {
	memberConfigDir := fmt.Sprintf("cluster/members/%v/config", m.MemberID)

	watcher := m.Config.DalClient.NewWatcher(memberConfigDir, true)

	m.Looper.Loop(func() error {
		resp, err := watcher.Next(context.Background())
		if err != nil {
			log.Errorf("%v: Unexpected watcher error: %v", m.Identifier, err.Error())
			return err
		}

		if m.ignorableWatcherEvent(resp) {
			log.Debugf("%v: Received an ignorable watcher '%v' event for key '%v'",
				m.Identifier, resp.Action, resp.Node.Key)
			return nil
		}

		log.Debugf("%v: Received a '%v' watcher event for '%v' (value: '%v')",
			m.Identifier, resp.Action, resp.Node.Key, resp.Node.Value)

		switch resp.Action {
		case "set":
			go m.Monitor.Handle(monitor.START, path.Base(resp.Node.Key), resp.Node.Value)
		case "delete":
			go m.Monitor.Handle(monitor.STOP, path.Base(resp.Node.Key), resp.Node.Value)
		default:
			log.Errorf("%v: Received an unrecognized action '%v' - skipping",
				m.Identifier, resp.Action)
			return fmt.Errorf("Unrecognized action '%v' on key %v", resp.Action, resp.Node.Key)
		}

		return nil
	})

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
