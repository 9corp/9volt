package director

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	etcd "github.com/coreos/etcd/client"

	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
)

const (
	COLLECT_CHECK_STATS_INTERVAL = time.Duration(60 * time.Second)
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

	CheckStats      map[string]int
	CheckStatsMutex *sync.Mutex
}

func New(cfg *config.Config, stateChan <-chan bool, distributeChan <-chan bool) (IDirector, error) {
	dalClient, err := dal.New(cfg.EtcdPrefix, cfg.EtcdMembers)
	if err != nil {
		return nil, err
	}

	return &Director{
		Identifier:      "director",
		Config:          cfg,
		MemberID:        cfg.MemberID,
		StateChan:       stateChan,
		DistributeChan:  distributeChan,
		StateLock:       &sync.Mutex{},
		DalClient:       dalClient,
		CheckStats:      make(map[string]int, 0),
		CheckStatsMutex: &sync.Mutex{},
	}, nil
}

func (d *Director) Start() error {
	log.Debugf("%v: Launching director components...", d.Identifier)

	go d.runDistributeListener()
	go d.runStateListener()
	go d.collectCheckStats()

	return nil
}

// This is used for figuring out how many checks are assigned to each member;
// this information is necessary for determining which member is next in line
// to be given/assigned a new check.
func (d *Director) collectCheckStats() {
	for {
		// To avoid a potential race here; all members have a count of how many
		// checks each member is assigned.
		// This will *probably* be switched to utilize `state` later on.
		checkStats, err := d.DalClient.FetchCheckStats()
		if err != nil {
			d.Config.EQClient.AddWithErrorLog("error",
				fmt.Sprintf("%v-collectCheckStats: Unable to fetch check stats: %v", d.Identifier, err.Error()))
			time.Sleep(COLLECT_CHECK_STATS_INTERVAL)
			continue
		}

		d.CheckStatsMutex.Lock()
		d.CheckStats = checkStats
		d.CheckStatsMutex.Unlock()

		time.Sleep(COLLECT_CHECK_STATS_INTERVAL)
	}
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
			d.Config.EQClient.AddWithErrorLog("error",
				fmt.Sprintf("%v-distributeListener: Unable to distribute checks: %v", d.Identifier, err.Error()))
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

	members2, err := d.DalClient.GetClusterMembersWithTags()
	if err != nil {
		log.Errorf("Unable to fetch members with tags: %v", err)
	}

	log.Warningf("Members2 contents: %v", members2)

	if len(members) == 0 {
		return fmt.Errorf("No active cluster members found - bug?")
	}

	log.Debugf("%v-distributeChecks: Distributing checks between %v cluster members", d.Identifier, len(members))

	// fetch all check keys
	checkKeys, err := d.DalClient.GetCheckKeys()
	if err != nil {
		return fmt.Errorf("Unable to fetch all check keys: %v", err.Error())
	}

	checkKeys2, err := d.DalClient.GetCheckKeysWithTags()
	if err != nil {
		log.Errorf("Unable to fetch check keys with tags: %v", err)
	}

	log.Warningf("Contents of checkKeys2: %v", checkKeys2)

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

			go d.runCheckConfigWatcher(ctx)

			// distribute checks in case we just took over as director (or first start)
			if err := d.distributeChecks(); err != nil {
				d.Config.EQClient.AddWithErrorLog("error",
					fmt.Sprintf("%v-stateListener: Unable to (re)distribute checks: %v", d.Identifier, err.Error()))
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

// Watch /monitor config changes so that we can update individual member configs
// ie. Something under /monitor changes -> figure out which member is responsible
//     for that particular check -> NOOP update OR DELETE corresponding member key
func (d *Director) runCheckConfigWatcher(ctx context.Context) {
	log.Debugf("%v-checkConfigWatcher: Launching...", d.Identifier)

	watcher := d.DalClient.NewWatcher("monitor/", true)

	// TODO: Needs to be turned into a looper
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
			log.Errorf("%v-checkConfigWatcher: Unexpected error: %v", d.Identifier, err.Error())
			continue
		}

		if d.ignorableWatcherEvent(resp) {
			log.Debugf("%v-checkConfigWatcher: Received ignorable watcher event for %v", d.Identifier, resp.Node.Key)
			continue
		}

		if err := d.handleCheckConfigChange(resp); err != nil {
			log.Errorf("%v-checkConfigWatcher: Unable to process config change for %v: %v",
				d.Identifier, resp.Node.Key, err.Error())
		}
	}

	log.Warningf("%v-checkConfigWatcher: Exiting...", d.Identifier)
}

func (d *Director) handleCheckConfigChange(resp *etcd.Response) error {
	log.Debugf("%v-handleCheckConfigChange: Received new response for key %v",
		d.Identifier, resp.Node.Key)

	memberRefs, err := d.DalClient.FetchAllMemberRefs()
	if err != nil {
		return fmt.Errorf("Unable to fetch all member refs: %v", err.Error())
	}

	log.Debugf("handleCheckConfigChange: contents of memberRefs: %v", memberRefs)

	var memberID string

	// If the check is brand new (ie. does not run on any node) - pick a node to run it on
	if val, ok := memberRefs[resp.Node.Key]; !ok {
		memberID = d.PickNextMember()
		log.Debugf("Check '%v' IS brand new; picked new node '%v' for it", resp.Node.Key, memberID)
	} else {
		log.Debugf("Check '%v' is NOT brand new and exists on member %v", resp.Node.Key, val)
		memberID = val
	}

	var actionErr error

	switch resp.Action {
	case "set":
		actionErr = d.DalClient.CreateCheckReference(memberID, resp.Node.Key)
	case "update":
		actionErr = d.DalClient.CreateCheckReference(memberID, resp.Node.Key)
	case "create":
		actionErr = d.DalClient.CreateCheckReference(memberID, resp.Node.Key)
	case "delete":
		actionErr = d.DalClient.ClearCheckReference(memberID, resp.Node.Key)
	default:
		return fmt.Errorf("%v-handleCheckConfigChange: Unrecognized action '%v'", d.Identifier, resp.Action)
	}

	if actionErr != nil {
		return fmt.Errorf("%v-handleCheckConfigChange: Unable to complete check config update: %v", d.Identifier, actionErr.Error())
	}

	return nil
}

// Return the least taxed cluster member
func (d *Director) PickNextMember() string {
	d.CheckStatsMutex.Lock()
	defer d.CheckStatsMutex.Unlock()

	if len(d.CheckStats) == 0 {
		// Check stats not yet populated, return self
		log.Debug("PickNextMember: Check stats are not yet populated, returning self")
		return d.MemberID
	}

	var leastTaxedMember string
	var leastChecks int

	for k, v := range d.CheckStats {
		// Handle first iteration
		if leastTaxedMember == "" {
			leastTaxedMember = k
			leastChecks = v
			continue
		}

		if v < leastChecks {
			leastTaxedMember = k
			leastChecks = v
		}
	}

	return leastTaxedMember
}

// Determine if a specific event can be ignored
func (d *Director) ignorableWatcherEvent(resp *etcd.Response) bool {
	if resp == nil {
		log.Debugf("%v: Received a nil etcd response - bug?", d.Identifier)
		return true
	}

	// Ignore `/monitor/`
	if path.Base(resp.Node.Key) == "monitor" {
		return true
	}

	return false
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
