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
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

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

	runDirectorMonitor()   // done
	runDirectorHeartbeat() // done
	runMemberMonitor()     // incomplete
	runMemberHeartbeat()   // incomplete

	getState() (*DirectorJSON, error)          // done
	handleState(*DirectorJSON) error           // done
	changeState(int, *DirectorJSON, int) error // done
	updateState(*DirectorJSON, int) error      // done
	isExpired(time.Time) bool                  // done
}

type Cluster struct {
	Config        *config.Config
	Identifier    string
	DirectorState bool
	DirectorLock  *sync.Mutex
	MemberID      string
	DalClient     dal.IDal // etcd client is/should-be thread safe
}

type DirectorJSON struct {
	MemberID   string
	LastUpdate time.Time
}

func New(cfg *config.Config) (ICluster, error) {
	dalClient, err := dal.New(cfg.EtcdPrefix, cfg.EtcdMembers)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		Config:        cfg,
		Identifier:    "cluster",
		DirectorState: false,
		DirectorLock:  new(sync.Mutex),
		MemberID:      util.GetMemberID(cfg.ListenAddress),
		DalClient:     dalClient,
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
		if err := c.sendHeartbeat(); err != nil {
			log.Errorf("%v-%v-directorHeartbeat: %v", err.Error())
		} else {
			log.Debugf("%v-%v-directorHeartbeat: Successfully sent periodic heartbeat",
				c.Identifier, c.MemberID)
		}
		time.Sleep(time.Duration(c.Config.HeartbeatInterval))
	}
}

func (c *Cluster) sendHeartbeat() error {
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

	for {
		if !c.amDirector() {
			// log.Debugf("%v-%v-memberMonitor: Not a director - nothing to do", c.Identifier, c.MemberID)
			time.Sleep(time.Duration(c.Config.HeartbeatInterval))
			continue
		}

		time.Sleep(time.Duration(c.Config.HeartbeatInterval))
	}
}

// ALWAYS: send member heartbeat updates
func (c *Cluster) runMemberHeartbeat() {
	log.Debugf("%v: Launching member heartbeat...", c.Identifier)

	for {
		// refresh our member dir; update (convenience) status blob
		time.Sleep(time.Duration(c.Config.HeartbeatInterval))
	}
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
