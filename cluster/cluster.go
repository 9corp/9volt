// Cluster engine package
//
// This package handles:
//
// 	 - check (re)distribution
//	 - director/member monitoring
//   - director/member heartbeats
//
// DirectorMonitor   - ALWAYS: monitor /cluster/director;
//                     inform DirectorHeartbeat to start (if current director dies)
//
// DirectorHeartbeat - IF DIRECTOR: send HeartbeatInterval updates to
//                     /cluster/director
//
// MemberMonitor     - IF DIRECTOR: monitor /cluster/members/; if new member_id
//					   appears (or gets removed) - inform director to redistribute
//                     checks
//
// MemberHeartbeat   - ALWAYS: send HeartbeatInterval updates to
//                     /cluster/members/member_id dir; send convenience
//                     status updates to /cluster/members/member_id/status
//

package cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/util"
)

const (
	// change state actions
	START int = iota
	STOP

	// etcd actions
	CREATE int = iota
	UPDATE
	NOOP

	DIRECTOR_KEY = "cluster/director"
)

type ICluster interface {
	Start() error

	// TODO: Implement go-director for loop control
	runDirectorMonitor()
	runDirectorHeartbeat()
	runMemberMonitor()
	runMemberHeartbeat()

	getState() (*DirectorJSON, error)
	handleState(*DirectorJSON) error
	changeState(int, *DirectorJSON, int) error
	updateState(*DirectorJSON, int) error
	isExpired(time.Time) bool
	amDirector() bool
	setDirectorState(bool)
	sendDirectorHeartbeat() error
}

type Cluster struct {
	Config         *config.Config
	Identifier     string
	DirectorState  bool
	DirectorLock   *sync.Mutex
	MemberID       string
	DalClient      dal.IDal // etcd client is/should-be thread safe
	Hostname       string
	StateChan      chan<- bool
	DistributeChan chan<- bool
}

type DirectorJSON struct {
	MemberID   string
	LastUpdate time.Time
}

type MemberJSON struct {
	MemberID      string
	Hostname      string
	ListenAddress string
	LastUpdated   time.Time
}

func New(cfg *config.Config, stateChan, distributeChan chan<- bool) (ICluster, error) {
	dalClient, err := dal.New(cfg.EtcdPrefix, cfg.EtcdMembers)
	if err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch hostname: %v", hostname)
	}

	return &Cluster{
		Config:         cfg,
		Identifier:     "cluster",
		DirectorState:  false,
		DirectorLock:   new(sync.Mutex),
		MemberID:       util.GetMemberID(cfg.ListenAddress),
		DalClient:      dalClient,
		Hostname:       hostname,
		StateChan:      stateChan,
		DistributeChan: distributeChan,
	}, nil
}

func (c *Cluster) Start() error {
	log.Debugf("%v: Launching cluster engine components...", c.Identifier)

	go c.runDirectorMonitor()
	go c.runDirectorHeartbeat()
	go c.runMemberMonitor()
	go c.runMemberHeartbeat()

	return nil
}

// ALWAYS: monitor /9volt/cluster/director to expire; become director
func (c *Cluster) runDirectorMonitor() {
	log.Debugf("%v: Launching director monitor...", c.Identifier)

	for {
		directorJSON, err := c.getState()
		if err != nil {
			log.Errorf("%v-%v-directorMonitor: Unable to fetch director state: %v",
				c.Identifier, c.MemberID, err.Error())
			time.Sleep(time.Duration(c.Config.HeartbeatInterval))
			continue
		}

		if err := c.handleState(directorJSON); err != nil {
			log.Errorf("%v-%v-directorMonitor: Unable to handle state: %v",
				c.Identifier, c.MemberID, err.Error())
		}

		time.Sleep(time.Duration(c.Config.HeartbeatInterval))
	}
}

// IF DIRECTOR: send periodic heartbeats to /9volt/cluster/director
func (c *Cluster) runDirectorHeartbeat() {
	log.Debugf("%v: Launching director heartbeat...", c.Identifier)

	for {
		if !c.amDirector() {
			// log.Debugf("%v-%v-directorHeartbeat: Not a director - nothing to do", c.Identifier, c.MemberID)
			time.Sleep(time.Duration(c.Config.HeartbeatInterval))
			continue
		}

		// update */director with current state data
		if err := c.sendDirectorHeartbeat(); err != nil {
			log.Errorf("%v-%v-directorHeartbeat: %v", err.Error())
		} else {
			log.Debugf("%v-%v-directorHeartbeat: Successfully sent periodic heartbeat",
				c.Identifier, c.MemberID)
		}
		time.Sleep(time.Duration(c.Config.HeartbeatInterval))
	}
}

func (c *Cluster) sendDirectorHeartbeat() error {
	newDirectorJSON := &DirectorJSON{
		MemberID:   c.MemberID,
		LastUpdate: time.Now(),
	}

	data, err := json.Marshal(newDirectorJSON)
	if err != nil {
		return fmt.Errorf("Unable to marshal heartbeat blob: %v", err.Error())
	}

	if err := c.DalClient.UpdateDirectorState(string(data), "", true); err != nil {
		return fmt.Errorf("Unable to update director heartbeat: %v", err.Error())
	}

	return nil
}

// IF DIRECTOR: monitor /9volt/cluster/members/*
func (c *Cluster) runMemberMonitor() {
	log.Debugf("%v: Launching member monitor...", c.Identifier)

	membersDir := "cluster/members/"

	// Create a watcher for cluster members
	watcher := c.DalClient.NewWatcher(membersDir, true)

	for {
		if !c.amDirector() {
			time.Sleep(time.Duration(c.Config.HeartbeatInterval))
			continue
		}

		// watch all dirs under /9volt/cluster/members/; alert if someone joins
		// or leaves
		resp, err := watcher.Next(context.Background())
		if err != nil {
			log.Errorf("%v-%v-memberMonitor: Unexpected watcher error: %v",
				c.Identifier, c.MemberID, err.Error())
			continue
		}

		switch resp.Action {
		case "set":
			// Only care about set's on base dir
			if !resp.Node.Dir {
				log.Debugf("%v-%v-memberMonitor: Ignoring watcher action on key %v",
					c.Identifier, c.MemberID, resp.Node.Key)
				continue
			}

			newMemberID := path.Base(resp.Node.Key)
			log.Infof("%v-%v-memberMonitor: New member '%v' has joined the cluster",
				c.Identifier, c.MemberID, newMemberID)
			c.DistributeChan <- true
		case "expire":
			// only dirs expire under /cluster/members/; don't need to do anything fancy
			oldMemberID := path.Base(resp.Node.Key)
			log.Infof("%v-%v-memberMonitor: Detected an expire for old member '%v'",
				c.Identifier, c.MemberID, oldMemberID)
			c.DistributeChan <- true
		default:
			continue
		}
	}
}

// Re-create member dir, set initial state
func (c *Cluster) createInitialMemberDir(memberDir string, heartbeatTimeoutInt int) error {
	// Pre-emptively remove potentially pre-existing memberdir and its children
	exists, _, err := c.DalClient.KeyExists(memberDir)
	if err != nil {
		return fmt.Errorf("Unable to verify pre-existence of member dir: %v", err.Error())
	}

	if exists {
		if err := c.DalClient.Delete(memberDir, true); err != nil {
			return fmt.Errorf("Unable to delete pre-existing member dir '%v': %v", memberDir, err.Error())
		}
	}

	// create initial dir
	if err := c.DalClient.Set(memberDir, "", true, heartbeatTimeoutInt, ""); err != nil {
		return fmt.Errorf("First member dir Set() failed: %v", err.Error())
	}

	// create initial member status
	memberJSON, err := c.generateMemberJSON()
	if err != nil {
		return fmt.Errorf("Unable to generate initial member JSON: %v",
			c.Config.HeartbeatInterval.String(), err.Error())
	}

	if err := c.DalClient.Set(memberDir+"/status", memberJSON, false, 0, ""); err != nil {
		return fmt.Errorf("Unable to create initial state: %v", err.Error())
	}

	return nil
}

// ALWAYS: send member heartbeat updates
// TODO: If we are not able to set the heartbeat - we should probably alert the
//       rest of 9volt components to shutdown or pause until we recover.
func (c *Cluster) runMemberHeartbeat() {
	log.Debugf("%v: Launching member heartbeat...", c.Identifier)

	memberDir := fmt.Sprintf("cluster/members/%v", c.MemberID)
	heartbeatTimeoutInt := int(time.Duration(c.Config.HeartbeatTimeout).Seconds())

	// create initial member dir
	if err := c.createInitialMemberDir(memberDir, heartbeatTimeoutInt); err != nil {
		log.Fatalf("%v-%v-memberHeartbeat: Unable to create initial member dir: %v",
			c.Identifier, c.MemberID, err.Error())
	}

	for {
		memberJSON, err := c.generateMemberJSON()
		if err != nil {
			log.Errorf("%v-%v-memberHeartbeat: Unable to generate member JSON (retrying in %v): %v",
				c.Identifier, c.MemberID, c.Config.HeartbeatInterval.String(), err.Error())
			time.Sleep(time.Duration(c.Config.HeartbeatInterval))
			continue
		}

		// set status key
		go func(memberDir, memberJSON string) {
			if err := c.DalClient.Set(memberDir+"/status", memberJSON, false, 0, "true"); err != nil {
				log.Errorf("%v-%v-memberHeartbeat: Unable to save member JSON status (retrying in %v): %v",
					c.Identifier, c.MemberID, c.Config.HeartbeatInterval.String(), err.Error())
			}
		}(memberDir, memberJSON)

		// refresh dir
		go func(memberDir string, ttl int) {
			if err := c.DalClient.Refresh(memberDir, heartbeatTimeoutInt); err != nil {
				// Not sure if this should cause a member dropout
				log.Errorf("%v-%v-memberHeartbeat: Unable to refresh member dir '%v' (retrying in %v): %v",
					memberDir, c.Config.HeartbeatInterval.String(), err.Error())
			}
		}(memberDir, heartbeatTimeoutInt)

		time.Sleep(time.Duration(c.Config.HeartbeatInterval))
	}
}

func (c *Cluster) generateMemberJSON() (string, error) {
	memberJSON := &MemberJSON{
		MemberID:      c.MemberID,
		Hostname:      c.Hostname,
		ListenAddress: c.Config.ListenAddress,
		LastUpdated:   time.Now(),
	}

	data, err := json.Marshal(memberJSON)
	if err != nil {
		return "", fmt.Errorf("Unable to marshal memberJSON: %v", err.Error())
	}

	return string(data), nil
}

func (c *Cluster) getState() (*DirectorJSON, error) {
	// Fetch the current state
	data, err := c.DalClient.Get(DIRECTOR_KEY, false)

	if c.DalClient.IsKeyNotFound(err) {
		log.Debugf("%v-%v-directorMonitor: No active director found", c.Identifier, c.MemberID)
		return nil, nil
	}

	if err != nil {
		log.Warningf("%v-%v-directorMonitor: Unexpected dal get error: %v",
			c.Identifier, c.MemberID, err.Error())
		return nil, err
	}

	// verify contents of director
	if _, ok := data[DIRECTOR_KEY]; !ok {
		return nil, fmt.Errorf("Uhh, no 'director' entry in map? Seems like a bug")
	}

	// Attempt to unmarshal
	var directorJSON DirectorJSON

	if err := json.Unmarshal([]byte(data[DIRECTOR_KEY]), &directorJSON); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal director JSON blob: %v", err.Error())
	}

	return &directorJSON, nil
}

func (c *Cluster) handleState(directorJSON *DirectorJSON) error {
	// nil directorJSON == no existing director entry
	if directorJSON == nil {
		log.Infof("%v-%v-directorMonitor: No existing director entry found - changing state!",
			c.Identifier, c.MemberID)
		return c.changeState(START, nil, CREATE)
	}

	// etcd says we are director, but we do not realize it
	// (ie. someone updated etcd manually and set us as director)
	if directorJSON.MemberID == c.MemberID {
		if !c.amDirector() {
			log.Infof("%v-%v-directorMonitor: Not a director, but etcd says we are (updating state)!",
				c.Identifier, c.MemberID)
			return c.changeState(START, nil, NOOP)
		}
	}

	// etcd says we are not director, but we think we are
	// (dealing with a potential race?)
	if directorJSON.MemberID != c.MemberID {
		if c.amDirector() {
			log.Warningf("%v-%v-directorMonitor: Running in director mode, but etcd says we are not!",
				c.Identifier, c.MemberID)
			return c.changeState(STOP, nil, NOOP)
		}
	}

	// happy path
	if directorJSON.MemberID != c.MemberID {
		if c.isExpired(directorJSON.LastUpdate) {
			log.Infof("%v-%v-directorMonitor: Current director '%v' expired; time to upscale!",
				c.Identifier, c.MemberID, directorJSON.MemberID)
			return c.changeState(START, directorJSON, UPDATE)
		} else {
			log.Infof("%v-%v-directorMonitor: Current director '%v' not expired yet; waiting patiently",
				c.Identifier, c.MemberID, directorJSON.MemberID)
		}
	}

	// Nothing happening
	return nil
}

func (c *Cluster) changeState(action int, prevDirectorJSON *DirectorJSON, etcdAction int) error {
	if action == START {
		log.Infof("%v-%v-directorMonitor: Taking over director role", c.Identifier, c.MemberID)

		// Only attempt to update state if we have to write to etcd (for UPDATE/CREATE)
		if etcdAction != NOOP {
			if err := c.updateState(prevDirectorJSON, etcdAction); err != nil {
				return fmt.Errorf("Unable to update director state: %v", err.Error())
			}
		}

		// Notify things to start? (ie. DirectorHeartbeat)
		c.setDirectorState(true)
	} else {
		log.Infof("%v-%v-directorMonitor: Shutting down director role", c.Identifier, c.MemberID)

		// Notify things to shutdown?
		c.setDirectorState(false)
	}

	return nil
}

func (c *Cluster) setDirectorState(newState bool) {
	c.DirectorLock.Lock()
	defer c.DirectorLock.Unlock()

	c.DirectorState = newState

	// Update state channel to inform director to start watching etcd
	c.StateChan <- newState
}

func (c *Cluster) updateState(prevDirectorJSON *DirectorJSON, etcdAction int) error {
	if etcdAction != CREATE && etcdAction != UPDATE {
		return fmt.Errorf("Unrecognized etcdAction '%v' (bug?)", etcdAction)
	}

	newDirectorJSON := &DirectorJSON{
		MemberID:   c.MemberID,
		LastUpdate: time.Now(),
	}

	data, err := json.Marshal(newDirectorJSON)
	if err != nil {
		return fmt.Errorf("Unable to marshal new director state blob: %v", err.Error())
	}

	var stateErr error
	var actionVerb string

	if etcdAction == UPDATE {
		// In order to compareAndSwap, we need to know the previous value
		prevData, marshalErr := json.Marshal(prevDirectorJSON)
		if marshalErr != nil {
			return fmt.Errorf("Unable to marshal previous director state data: %v", err.Error())
		}

		stateErr = c.DalClient.UpdateDirectorState(string(data), string(prevData), false)
		actionVerb = "update"
	} else {
		stateErr = c.DalClient.CreateDirectorState(string(data))
		actionVerb = "create"
	}

	if stateErr != nil {
		return fmt.Errorf("Unable to %v director state in dal: %v", actionVerb, stateErr.Error())
	}

	log.Debugf("%v-%v-directorMonitor: Successfully %vd director state in dal",
		c.Identifier, c.MemberID, actionVerb)

	return nil
}

func (c *Cluster) isExpired(lastUpdated time.Time) bool {
	delta := time.Now().Sub(lastUpdated)

	if delta.Seconds() > time.Duration(c.Config.HeartbeatTimeout).Seconds() {
		return true
	}

	return false
}

func (c *Cluster) amDirector() bool {
	c.DirectorLock.Lock()
	defer c.DirectorLock.Unlock()

	// hmm - can we just `return c.DirectorState` here?
	if c.DirectorState {
		return true
	}

	return false
}
