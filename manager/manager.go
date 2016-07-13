package manager

import (
	log "github.com/Sirupsen/logrus"
	// "golang.org/x/net/context"

	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/util"
)

type IManager interface {
	Start() error
}

type Manager struct {
	DalClient  dal.IDal // etcd client is/should-be thread safe
	MemberID   string
	Identifier string
}

func New(cfg *config.Config) (IManager, error) {
	dalClient, err := dal.New(cfg.EtcdPrefix, cfg.EtcdMembers)
	if err != nil {
		return nil, err
	}

	return &Manager{
		Identifier: "manager",
		DalClient:  dalClient,
		MemberID:   util.GetMemberID(cfg.ListenAddress),
	}, nil
}

func (m *Manager) Start() error {
	go m.runConfigMonitor()

	return nil
}

func (m *Manager) runConfigMonitor() {
	log.Debugf("%v: Launching configuration manager...", m.Identifier)
}

// TODO: If our 'member dir' disappears, should we stop all monitors?
// TODO: Does a state change mean we should cease monitoring?
