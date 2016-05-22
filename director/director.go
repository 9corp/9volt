package director

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"

	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/util"
)

type IDirector interface {
	Start() error
}

type Director struct {
	Identifier     string
	MemberID       string
	Config         *config.Config
	State          bool
	StateChan      <-chan bool
	DistributeChan <-chan bool
	StateLock      *sync.Mutex
	DalClient      dal.IDal
}

func New(cfg *config.Config, stateChan <-chan bool, distributeChan <-chan bool) (IDirector, error) {
	dalClient, err := dal.New(cfg.EtcdPrefix, cfg.EtcdMembers)
	if err != nil {
		return nil, err
	}

	return &Director{
		Identifier:     "director",
		MemberID:       util.GetMemberID(cfg.ListenAddress),
		Config:         cfg,
		StateChan:      stateChan,
		DistributeChan: distributeChan,
		StateLock:      &sync.Mutex{},
		DalClient:      dalClient,
	}, nil
}

func (d *Director) Start() error {
	log.Debugf("%v: Launching director components...", d.Identifier)

	go d.runDistributeListener()
	go d.runStateListener()

	return nil
}

func (d *Director) runDistributeListener() {
	for {
		<-d.DistributeChan

		// safety valve
		if !d.amDirector() {
			log.Warningf("%v-distributeListener: Was asked to distribute checks but am not director!", d.Identifier)
			continue
		}

		if err := d.distributeChecks(); err != nil {
			log.Errorf("%v-distributeListener: Unable to distribute checks: %v", d.Identifier, err.Error())
		}
	}
}

func (d *Director) distributeChecks() error {
	log.Infof("%v-distributeChecks: Performing check distribution across members in cluster", d.Identifier)

	// fetch all members in cluster
	members, err := d.DalClient.GetClusterMembers()
	if err != nil {
		return fmt.Errorf("Unable to fetch cluster members: %v", err.Error())
	}

	log.Debugf("%v-distributeChecks: Distributing checks between %v cluster members", len(members))

	// fetch all check keys
	checkKeys, error := d.DalClient.GetCheckKeys()
	if err != nil {
		return fmt.Errorf("Unable to fetch all check keys: %v", err.Error())
	}

	if len(checkKeys) == 0 {
		return fmt.Errorf("Check configuration is empty - nothing to distribute!")
	}

	if err := d.performCheckDistribution(members, checkKeys); err != nil {
		return fmt.Errorf("Unable to complete check distribution: %v", err.Error())
	}

	return nil
}

// A simple (and equal) check distributor
//
// Divide checks equally between all members; last member gets remainder of checks
func (d *Director) performCheckDistribution(members, checkKey []string) error {
	checksPerMember := len(checkKeys) / len(members)

	memberNum := 0
	checkKeyNum := 0

	for memberNum := 0; memberNum < len(members); memberNum++ {
		maxChecks := checkKeyNum + checksPerMember

		// last member gets the remainder of the checks
		if memberNum == len(members)-1 {
			maxChecks = len(checkKeys)
		}

		totalAssigned := 0

		for i := checkKeyNum; i != maxChecks; i++ {
			totalAssigned++
		}

		log.Debugf("%v-distributeChecks: Assigned %v checks to %v", d.Identifier,
			totalAssigned, member)
	}

	return nil
}

func (d *Director) runStateListener() {
	var ctx context.Context
	var cancel context.CancelFunc

	for {

		state := <-d.StateChan

		if state {
			log.Infof("%v-stateListener: Starting up etcd watchers", d.Identifier)

			// create new context + cancel func
			ctx, cancel = context.WithCancel(context.Background())

			// distribute checks in case we just took over as director (or first start)
			if err := d.distributeChecks(); err != nil {
				log.Errorf("%v-stateListener: Unable to distribute checks: %v", d.Identifier, err.Error())
			}

			go d.runHostConfigWatcher(ctx)
			go d.runCheckConfigWatcher(ctx)
		} else {
			log.Infof("%v-stateListener: Shutting down etcd watchers", d.Identifier)
			cancel()
		}

		d.setState(state)
	}
}

func (d *Director) runHostConfigWatcher(ctx context.Context) {
	watcher := d.DalClient.NewWatcher("host/", true)

	for {
		// safety valve
		if !d.amDirector() {
			log.Warningf("%v-hostConfigWatcher: Not active director - stopping", d.Identifier)
			break
		}

		// watch host config entries
		resp, err := watcher.Next(ctx)
		if err != nil && err.Error() == "context canceled" {
			log.Warningf("%v-hostConfigWatcher: Received a notice to shutdown", d.Identifier)
			break
		} else if err != nil {
			log.Errorf("%v-hostConfigWatcher: Unexpected error: %v", err.Error())
			continue
		}

		if err := d.handleHostConfigChange(resp); err != nil {
			log.Errorf("%v-hostConfigWatcher: Unable to process config change for %v: %v",
				d.Identifier, resp.Node.Key, err.Error())
		}
	}

	log.Warningf("%v-hostConfigWatcher: Exiting...", d.Identifier)
}

func (d *Director) runCheckConfigWatcher(ctx context.Context) {
	watcher := d.DalClient.NewWatcher("monitor/", true)

	for {
		// safety valve
		if !d.amDirector() {
			log.Warningf("%v-checkConfigWatcher: Not active director - stopping", d.Identifier)
			break
		}

		// watch check config entries
		resp, err := watcher.Next(ctx)
		if err != nil && err.Error() == "context canceled" {
			log.Warningf("%v-checkConfigWatcher: Received a notice to shutdown", d.Identifier)
			break
		} else if err != nil {
			log.Errorf("%v-checkConfigWatcher: Unexpected error: %v", err.Error())
			continue
		}

		if err := d.handleCheckConfigChange(resp); err != nil {
			log.Errorf("%v-hostConfigWatcher: Unable to process config change for %v: %v",
				d.Identifier, resp.Node.Key, err.Error())
		}
	}

	log.Warningf("%v-checkConfigWatcher: Exiting...", d.Identifier)
}

func (d *Director) handleCheckConfigChange(resp *etcd.Response) error {
	log.Debugf("%v-handleCheckConfigChange: Received new response for key %v",
		d.Identifier, resp.Node.Key)

	return nil
}

func (d *Director) handleHostConfigChange(resp *etcd.Response) error {
	log.Debugf("%v-handleHostConfigChange: Received new response for key %v",
		d.Identifier, resp.Node.Key)

	return nil
}

func (d *Director) setState(state bool) {
	d.StateLock.Lock()
	d.State = state
	d.StateLock.Unlock()
}

func (d *Director) amDirector() bool {
	d.StateLock.Lock()
	state := d.State
	d.StateLock.Unlock()

	return state
}
