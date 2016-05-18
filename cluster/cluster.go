// Cluster engine package
//
// This package handles:
//
// 	 - check (re)distribution
//	 - director/member monitoring
//   - director/member heartbeats
//
// DirectorMonitor   - IF NOT DIRECTOR: monitor /cluster/director;
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
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
)

const (
	RETRY_INTERVAL = time.Duration(5) * time.Second
)

type ICluster interface {
	Start() error

	runDirectorMonitor()
	runDirectorHeartbeat()
	runMemberMonitor()
	runMemberHeartbeat()
	amDirector() bool
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

// IF NOT DIRECTOR: monitor /9volt/cluster/director to expire; become director
func (c *Cluster) runDirectorMonitor() {
	log.Debugf("%v: Launching director monitor...", c.Identifier)

	dalClient, err := dal.New(c.Config.EtcdPrefix, c.Config.EtcdMembers)

	if err != nil {
		log.Fatalf("%v-directorMonitor: Unable to start due to dal client error: %v",
			c.Identifier, err.Error())
	}

	for {
		if c.amDirector() {
			log.Debugf("%v-%v-directorMonitor: Current director - no need to monitor endpoint", c.Identifier, c.MemberID)
			time.Sleep(RETRY_INTERVAL)
			continue
		}

		// Verify contents of '/cluster/director', becomeDirector (maybe)
		data, err := dalClient.Get("cluster/members/director")

		if dalClient.IsKeyNotFound(err) {
			log.Infof("%v-%v-directorMonitor: No director found - upscaling ourselves!", c.Identifier, c.MemberID)

			if err := c.becomeDirector(); err != nil {
				log.Errorf("%v-%v-directorMonitor: Unable to become director: %v", c.Identifier, c.MemberID)
				time.Sleep(RETRY_INTERVAL)
				continue
			}
		}

		// verify contents of director
		if _, ok := data["director"]; !ok {
			log.Errorf("%v-%v-directorMonitor: Uhh, no 'director' in map? Seems like a bug", c.Identifier, c.MemberID)
			time.Sleep(RETRY_INTERVAL)
			continue
		}

		var directorJSON DirectorJSON

		if err := json.Unmarshal([]byte(data["director"], &directorJSON)); err != nil {
			log.Errorf("%v-%v-directorMonitor: Unable to unmarshal director JSON blob: %v",
				c.Identifier, c.MemberID, err.Error())

			time.Sleep(RETRY_INTERVAL)
			continue
		}

		// validate contents of director json blob
		if err := c.validateDirectorJSON(*directorJSON); err != nil {
			log.Errorf("%v-%v-directorMonitor: Unable to validate director JSON blob: %v", c.Identifier, c.MemberID, err.Error())
			time.Sleep(RETRY_INTERVAL)
			continue
		}

		// check if we are expired
		if c.isExpired(directorJson.LastUpdated) {
			log.Debugf("%v-%v-directorMonitor: timestamp expired in current director blob -> upscaling ourselves!", c.Identifier, c.MemberID)

			if err := c.becomeDirector(); err != nil {
				log.Errorf("%v-%v-directorMonitor: Unable to become director (case 2): %v", c.Identifier, c.MemberID, err.Error())
				time.Sleep(RETRY_INTERVAL)
				continue
			}
		}

		log.Debugf("%v-%v-directorMonitor: Current director (%v) still alive - nothing to do", c.Identifier, c.MemberID, directorJSON.MemberID)
		time.Sleep(RETRY_INTERVAL)
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

func (c *Cluster) amDirector() bool {
	c.DirectorLock.Lock()
	defer c.DirectorLock.Unlock()

	// hmm - can we just `return c.DirectorState` here?
	if c.DirectorState {
		return true
	}

	return false
}
