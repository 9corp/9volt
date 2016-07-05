package director

import (
	"fmt"
	"sync"
	"time"

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
		// Notification sent by cluster component
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
	log.Debugf("%v-distributeChecks: Performing member existence verification", d.Identifier)

	if err := d.verifyMemberExistence(); err != nil {
		return fmt.Errorf("%v-distributeChecks: Unable to verify member existence in cluster: %v",
			d.Identifier, err.Error())
	}

	log.Infof("%v-distributeChecks: Performing check distribution across members in cluster", d.Identifier)

	// fetch all members in cluster
	members, err := d.DalClient.GetClusterMembers()
	if err != nil {
		return fmt.Errorf("Unable to fetch cluster members: %v", err.Error())
	}

	if len(members) == 0 {
		return fmt.Errorf("No active cluster members found - bug?")
	}

	log.Debugf("%v-distributeChecks: Distributing checks between %v cluster members", d.Identifier, len(members))

	// fetch all check keys
	checkKeys, err := d.DalClient.GetCheckKeys()
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
func (d *Director) performCheckDistribution(members, checkKeys []string) error {
	checksPerMember := len(checkKeys) / len(members)

	start := 0

	for memberNum := 0; memberNum < len(members); memberNum++ {
		// Blow away any pre-existing config references
		if err := d.DalClient.ClearCheckReferences(members[memberNum]); err != nil {
			log.Errorf("%v: Unable to clear existing check references for member %v: %v",
				d.Identifier, members[memberNum], err.Error())
			return err
		}

		maxChecks := start + checksPerMember

		// last member gets the remainder of the checks
		if memberNum == len(members)-1 {
			maxChecks = len(checkKeys)
		}

		totalAssigned := 0

		for i := start; i != maxChecks; i++ {
			log.Debugf("%v: Assigning check '%v' to member '%v'", d.Identifier, checkKeys[i], members[memberNum])

			if err := d.DalClient.CreateCheckReference(members[memberNum], checkKeys[i]); err != nil {
				log.Errorf("%v: Unable to create check reference for member %v: %v",
					d.Identifier, members[memberNum], err.Error())
				return err
			}

			totalAssigned++
		}

		// Update our start num
		start = maxChecks

		log.Debugf("%v-distributeChecks: Assigned %v checks to %v", d.Identifier,
			totalAssigned, members[memberNum])
	}

	return nil
}

func (d *Director) runStateListener() {
	var ctx context.Context
	var cancel context.CancelFunc

	for {
		state := <-d.StateChan

		d.setState(state)

		if state {
			log.Infof("%v-stateListener: Starting up etcd watchers", d.Identifier)

			// create new context + cancel func
			ctx, cancel = context.WithCancel(context.Background())

			go d.runHostConfigWatcher(ctx)
			go d.runCheckConfigWatcher(ctx)
			go d.runAlertConfigWatcher(ctx)

			// distribute checks in case we just took over as director (or first start)
			if err := d.distributeChecks(); err != nil {
				log.Errorf("%v-stateListener: Unable to (re)distribute checks: %v", d.Identifier, err.Error())
			}
		} else {
			log.Infof("%v-stateListener: Shutting down etcd watchers", d.Identifier)
			cancel()
		}
	}
}

// This method exists to deal with a case where a director launches for the
// first time and attempts to distribute checks but the memberHeartbeat() has not
// yet had a chance to populate itself under /cluster/members/*
func (d *Director) verifyMemberExistence() error {
	// TODO: This can probably go into dal.GetClusterMembers()

	// Let's wait a `heartbeatInterval`*2 to ensure that at least 1 active member
	// is in the cluster (if not - there's either a bug or the system is *massively* overloaded)
	tmpCtx, _ := context.WithTimeout(context.Background(), time.Duration(d.Config.HeartbeatInterval)*2)
	tmpWatcher := d.DalClient.NewWatcher("cluster/members/", true)

	for {
		resp, err := tmpWatcher.Next(tmpCtx)
		if err != nil {
			return fmt.Errorf("Error waiting on /cluster/members/*: %v", err.Error())
		}

		if resp.Action != "set" && resp.Action != "update" {
			log.Debugf("%v-verifyMemberExistence: Ignoring '%v' action on key %v",
				d.Identifier, resp.Action, resp.Node.Key)
			continue
		}

		log.Debugf("%v-verifyMemberExistence: Detected '%v' action for key %v",
			d.Identifier, resp.Action, resp.Node.Key)

		return nil
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

// Watch /monitor config changes so that we can update individual member configs
// ie. Something under /monitor changes -> figure out which member is responsible
//     for that particular check -> NOOP update OR DELETE corresponding member key
func (d *Director) runCheckConfigWatcher(ctx context.Context) {
	log.Debugf("%v-checkConfigWatcher: Launching...", d.Identifier)

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

func (d *Director) runAlertConfigWatcher(ctx context.Context) {
	log.Debugf("%v-alertConfigWatcher: Launching...", d.Identifier)

	watcher := d.DalClient.NewWatcher("alert/", true)

	for {
		// safety valve
		if !d.amDirector() {
			log.Warningf("%v-alertConfigWatcher: Not active director - stopping", d.Identifier)
			break
		}

		// watch check config entries
		resp, err := watcher.Next(ctx)
		if err != nil && err.Error() == "context canceled" {
			log.Warningf("%v-alertConfigWatcher: Received a notice to shutdown", d.Identifier)
			break
		} else if err != nil {
			log.Errorf("%v-alertConfigWatcher: Unexpected error: %v", err.Error())
			continue
		}

		if err := d.handleAlertConfigChange(resp); err != nil {
			log.Errorf("%v-hostConfigWatcher: Unable to process config change for %v: %v",
				d.Identifier, resp.Node.Key, err.Error())
		}
	}

	log.Warningf("%v-alertConfigWatcher: Exiting...", d.Identifier)
}

// TODO: Update '/9volt/cluster/members/member_id/config/*' entry
func (d *Director) handleCheckConfigChange(resp *etcd.Response) error {
	log.Debugf("%v-handleCheckConfigChange: Received new response for key %v",
		d.Identifier, resp.Node.Key)

	return nil
}

// TODO: Update '/9volt/cluster/members/member_id/config/*' entry
func (d *Director) handleAlertConfigChange(resp *etcd.Response) error {
	log.Debugf("%v-handleAlertConfigChange: Received new response for key %v",
		d.Identifier, resp.Node.Key)

	return nil
}

// TODO: Update '/9volt/cluster/members/member_id/config/*' entry
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
