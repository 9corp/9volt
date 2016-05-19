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
)

const (
	RETRY_INTERVAL = time.Duration(5) * time.Second

	START int = iota
	STOP
)

type ICluster interface {
	Start() error

	runDirectorMonitor()   // incomplete
	runDirectorHeartbeat() // incomplete
	runMemberMonitor()     // incomplete
	runMemberHeartbeat()   // incomplete
}

type Cluster struct {
	Config        *config.Config
	Identifier    string
	DirectorState bool
	DirectorLock  sync.Mutex
	MemberID      string
}

type DirectorJSON struct {
	MemberID   string
	LastUpdate time.Time
}

func New(cfg *config.Config) ICluster {
	return &Cluster{
		Config:        cfg,
		Identifier:    "cluster",
		DirectorState: false,
		DirectorLock:  new(sync.Mutex),
		MemberID:      util.GetMemberID(),
	}
}

func (c *Cluster) Start() error {
	log.Debugf("%v: Launching things...", c.Identifier)

	go c.runDirectorMonitor()
	go c.runDirectorHeartbeat()
	go c.runMemberMonitor()
	go c.runMemberHeartbeat()

	return nil
}

// ALWAYS: monitor /9volt/cluster/director to expire; become director
func (c *Cluster) runDirectorMonitor() {
	log.Debugf("%v: Launching director monitor...", c.Identifier)

	// Test if we are able to create a dal client
	_, err := dal.New(c.Config.EtcdPrefix, c.Config.EtcdMembers)
	if err != nil {
		log.Fatalf("%v-%v-directorMonitor: Unable to start due to dal client error: %v",
			c.Identifier, c.MemberID, err.Error())
	}

	for {
		directorJSON, err := c.getState()
		if err != nil {
			log.Errorf("%v-%v-directorMonitor: Unable to fetch director state: %v",
				c.Identifier, c.MemberID, err.Error())
			time.Sleep(RETRY_INTERVAL)
			continue
		}

		if err := c.handleState(directorJSON); err != nil {
			log.Errorf("%v-%v-directorMonitor: Unable to handle state: %v",
				c.Identifier, c.MemberID, err.Error())
		}
	}
}

// IF DIRECTOR: send periodic heartbeats to /9volt/cluster/director
func (c *Cluster) runDirectorHeartbeat() {
	log.Debugf("%v: Launching director heartbeat...", c.Identifier)

	for {
		if !c.amDirector() {
			log.Debugf("%v-directorHeartbeat: Not a director - nothing to do", c.Identifier)
			time.Sleep(RETRY_INTERVAL)
			continue
		}

		// update */director with current state data

		time.Sleep(time.Duration(c.Config.HeartbeatInterval))
	}
}

// IF DIRECTOR: monitor /9volt/cluster/members/*
func (c *Cluster) runMemberMonitor() {
	log.Debugf("%v: Launching member monitor...", c.Identifier)

	for {
		if !c.amDirector() {
			log.Debugf("%v-memberMonitor: Not a director - nothing to do", c.Identifier)
			time.Sleep(RETRY_INTERVAL)
			continue
		}

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
	// Verify contents of '/cluster/director', becomeDirector (maybe)
	dalClient, err := dal.New(c.Config.EtcdPrefix, c.Config.EtcdMembers)
	if err != nil {
		return nil, err
	}

	// Fetch the current state
	data, err := dalClient.Get("cluster/members/director")

	if dalClient.IsKeyNotFound(err) {
		log.Debugf("%v-%v-directorMonitor: No active director found", c.Identifier, c.MemberID)
		return nil, nil
	}

	// verify contents of director
	if _, ok := data["director"]; !ok {
		return nil, fmt.Errorf("Uhh, no 'director' entry in map? Seems like a bug")
	}

	// Attempt to unmarshal
	var directorJSON DirectorJSON

	if err := json.Unmarshal([]byte(data["director"], &directorJSON)); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal director JSON blob: %v", err.Error())
	}

	return &directorJSON, nil
}

func (c *Cluster) handleState(directorJSON *DirectorJSON) error {
	changeState := false

	// nil directorJSON == no existing director entry
	if directorJSON == nil {
		log.Infof("%v-%v-directorMonitor: No existing director entry found - changing state!",
			c.Identifier, c.MemberID)
		return c.changeState(START)
	}

	// etcd says we are director, but we do not realize it
	// (ie. someone updated etcd manually and set us as director)
	if directorJSON.MemberID == c.MemberID {
		if !c.amDirector() {
			log.Infof("%v-%v-directorMonitor: Not a director, but etcd says we are!",
				c.Identifier, c.MemberID)
			return c.changeState(START)
		}
	}

	// etcd says we are not director, but we think we are
	// (dealing with a potential race?)
	if directorJSON.MemberID != c.MemberID {
		if c.amDirector() {
			log.Warningf("%v-%v-directorMonitor: Running in director mode, but etcd says we are not!",
				c.Identifier, c.MemberID)
			return c.changeState(STOP)
		}
	}

	// happy path
	if directorJSON.MemberID != c.MemberID && c.isExpired(directorJSON.LastUpdate) {
		log.Infof("%v-%v-directorMonitor: Current director '%v' expired; time to upscale!",
			c.Identifier, c.MemberID, directorJSON.MemberID)
		return c.changeState(START)
	}

	// Nothing happening
	return nil
}

func (c *Cluster) changeState(action int) error {
	if START {
		if err := c.updateState(); err != nil {
			return fmt.Errorf("Unable to update director state: %v", err.Error())
		}

		// Notify things to start?
		c.setDirectorState(true)
	} else {
		// notify things to shut down?
		c.setDirectorState(false)
	}

	return nil
}

func (c *Cluster) updateState() error {
	dalClient, err := dal.New(c.Config.EtcdPrefix, c.Config.EtcdMembers)
	if err != nil {
		return fmt.Errorf("Unable to instantiate dal client for state change: %v", err.Error())
	}

	directorJSON := &DirectorJSON{
		MemberID:   c.MemberID,
		LastUpdate: time.Now(),
	}

	data, err := json.Marshal(directorJSON)
	if err != nil {
		return fmt.Errorf("Unable to marshal director state blob: %v", err.Error())
	}

	if err := dalClient.UpdateDirectorState(string(data)); err != nil {
		return fmt.Errorf("Unable to update director state in dal: %v", err.Error())
	}

	log.Debugf("%v-%v-directorMonitor: Successfully updated director state in dal",
		c.Identifier, c.MemberID)

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
