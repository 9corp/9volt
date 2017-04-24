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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	looper "github.com/relistan/go-director"

	"github.com/9corp/9volt/base"
	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/overwatch"
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

	runDirectorMonitor()
	runDirectorHeartbeat()
	runMemberHeartbeat()
	runMemberMonitor()

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
	Config                  *config.Config
	Log                     log.FieldLogger
	DirectorState           bool
	DirectorLock            *sync.Mutex
	MemberID                string
	DalClient               dal.IDal // etcd client is/should-be thread safe
	Hostname                string
	StateChan               chan<- bool
	DistributeChan          chan<- bool
	OverwatchChan           chan<- *overwatch.Message
	initFinished            chan bool
	DirectorMonitorLooper   looper.Looper
	DirectorHeartbeatLooper looper.Looper
	MemberHeartbeatLooper   looper.Looper
	restarted               bool
	shutdown                bool // provide a way for basic (non-director) loopers to exit

	base.Component
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
	Tags          []string
	Version       string
	SemVer        string
}

var (
	// overwrite for tests
	logFatal = func(logger log.FieldLogger, fields log.Fields, msg string) {
		logger.WithFields(fields).Fatal(msg)
	}
)

func New(cfg *config.Config, stateChan, distributeChan chan<- bool, overwatchChan chan<- *overwatch.Message) (*Cluster, error) {
	dalClient, err := dal.New(cfg.EtcdPrefix, cfg.EtcdMembers, cfg.EtcdUserPass, false, false, false)
	if err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch hostname: %v", hostname)
	}

	return &Cluster{
		Config:                  cfg,
		Log:                     log.WithField("pkg", "cluster"),
		DirectorState:           false,
		DirectorLock:            new(sync.Mutex),
		MemberID:                cfg.MemberID,
		DalClient:               dalClient,
		Hostname:                hostname,
		StateChan:               stateChan,
		DistributeChan:          distributeChan,
		OverwatchChan:           overwatchChan,
		DirectorMonitorLooper:   looper.NewImmediateTimedLooper(looper.FOREVER, time.Duration(cfg.HeartbeatInterval), make(chan error, 1)),
		DirectorHeartbeatLooper: looper.NewImmediateTimedLooper(looper.FOREVER, time.Duration(cfg.HeartbeatInterval), make(chan error, 1)),
		MemberHeartbeatLooper:   looper.NewImmediateTimedLooper(looper.FOREVER, time.Duration(cfg.HeartbeatInterval), make(chan error, 1)),
		initFinished:            make(chan bool, 1),
		Component: base.Component{
			Identifier: "cluster",
		},
	}, nil
}

func (c *Cluster) Start() error {
	c.Log.Info("Launching cluster engine components...")

	c.Component.Ctx, c.Component.Cancel = context.WithCancel(context.Background())
	c.shutdown = false

	go c.runDirectorMonitor()
	go c.runDirectorHeartbeat()

	// memberHeartbeat creates initial member structure; wait until that's
	// completed before starting the memberMonitor or otherwise we may run into
	// a race
	go c.runMemberHeartbeat()

	<-c.initFinished

	go c.runMemberMonitor()

	return nil
}

func (c *Cluster) Stop() error {
	// stop the director monitor
	c.DirectorMonitorLooper.Quit()

	// stop the director heartbeat send
	c.DirectorHeartbeatLooper.Quit()

	// alert memberMonitor to stop if we're not a director
	c.shutdown = true

	// stop memberMonitor
	if c.Component.Cancel == nil {
		c.Log.Warning("Looks like .Cancel is nil; is this expected?")
	} else {
		c.Component.Cancel()
	}

	// stop memberHeartbeat
	c.MemberHeartbeatLooper.Quit()

	// Let the subcomponents know that we've been
	c.restarted = true

	return nil
}

// ALWAYS: monitor /9volt/cluster/director to expire; become director
func (c *Cluster) runDirectorMonitor() {
	llog := c.Log.WithField("method", "runDirectorMonitor")

	llog.Debug("Launching director monitor...")

	c.DirectorMonitorLooper.Loop(func() error {
		directorJSON, err := c.getState()
		if err != nil {
			c.Config.EQClient.AddWithErrorLog("error", "Unable to fetch director state", llog, log.Fields{"err": err})
			return nil
		}

		if err := c.handleState(directorJSON); err != nil {
			c.Config.EQClient.AddWithErrorLog("error", "Unable to handle state", llog, log.Fields{"err": err})
		}

		return nil
	})

	llog.Debug("Exiting...")
}

// IF DIRECTOR: send periodic heartbeats to /9volt/cluster/director
func (c *Cluster) runDirectorHeartbeat() {
	llog := c.Log.WithField("method", "runDirectorHeartbeat")

	llog.Debug("Launching director heartbeat...")

	c.DirectorHeartbeatLooper.Loop(func() error {
		if !c.amDirector() {
			// log.Debugf("%v-directorHeartbeat: Not a director - nothing to do", c.Identifier)
			return nil
		}

		// update */director with current state data
		if err := c.sendDirectorHeartbeat(); err != nil {
			c.Config.EQClient.AddWithErrorLog("error", "Unable to send director heartbeat", llog, log.Fields{"err": err})

			// Let overwatch decide what to do in this case
			c.OverwatchChan <- &overwatch.Message{
				Error:     fmt.Errorf("Potential etcd write error: %v", err),
				Source:    fmt.Sprintf("%v.runDirectorHeartbeat", c.Identifier),
				ErrorType: overwatch.ETCD_GENERIC_ERROR,
			}

			return nil
		} else {
			llog.WithField("id", c.MemberID).Debug("Successfully sent periodic heartbeat")
		}

		return nil
	})

	llog.Debug("Exiting...")
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
	llog := c.Log.WithField("method", "runMemberMonitor")

	llog.Debug("Launching member monitor...")

	membersDir := "cluster/members/"

	// Create a watcher for cluster members
	watcher := c.DalClient.NewWatcher(membersDir, true)

	for {
		// This for loop cannot be a director looper -- happy path is to block
		// indefinitely on the watch - with a director looper, another execution
		// would be triggered when the interval is reached.
		//
		// This could be a channel, but this seems easy albeit a bit hacky
		if c.shutdown {
			llog.Debug("Received a (non-context based) notice to shutdown")
			break
		}

		if !c.amDirector() {
			time.Sleep(time.Duration(c.Config.HeartbeatInterval))
			continue
		}

		// watch all dirs under /9volt/cluster/members/; alert if someone joins
		// or leaves
		resp, err := watcher.Next(c.Component.Ctx)
		if err != nil {
			if err.Error() == "context canceled" {
				llog.Debug("Received a notice to shutdown")
				break
			}

			c.Config.EQClient.AddWithErrorLog("error", "Unexpected watcher error", llog, log.Fields{"err": err})

			c.OverwatchChan <- &overwatch.Message{
				Error:     fmt.Errorf("Watcher error: %v", err),
				Source:    fmt.Sprintf("%v.runMemberMonitor", c.Identifier),
				ErrorType: overwatch.ETCD_WATCHER_ERROR,
			}

			// Let overwatch determine if it should shut things down or not
			continue
		}

		switch resp.Action {
		case "set":
			// Only care about set's on base dir and 'config'
			if !resp.Node.Dir || path.Base(resp.Node.Key) == "config" {
				llog.WithField("key", resp.Node.Key).Debug("Ignoring watcher action")
				continue
			}

			newMemberID := path.Base(resp.Node.Key)
			llog.WithField("member", newMemberID).Info("New member has joined the cluster")
			c.DistributeChan <- true
		case "expire":
			// only dirs expire under /cluster/members/; don't need to do anything fancy
			oldMemberID := path.Base(resp.Node.Key)
			llog.WithField("member", oldMemberID).Info("Detected an expire for old member")
			c.DistributeChan <- true
		default:
			continue
		}
	}

	llog.Debug("Exiting...")
}

// Re-create member dir structure, set initial state
func (c *Cluster) createInitialMemberStructure(memberDir string, heartbeatTimeoutInt int) error {
	// Pre-emptively remove potentially pre-existing memberdir and its children
	exists, _, err := c.DalClient.KeyExists(memberDir)
	if err != nil {
		return fmt.Errorf("Unable to verify pre-existence of member dir: %v", err.Error())
	}

	if exists {
		c.Log.Infof("MemberDir %v already exists. Trying to delete...", memberDir)

		if err := c.DalClient.Delete(memberDir, true); err != nil {
			return fmt.Errorf("Unable to delete pre-existing member dir '%v': %v", memberDir, err.Error())
		}
	}

	// create initial dir
	if err := c.DalClient.Set(memberDir, "", &dal.SetOptions{Dir: true, TTLSec: heartbeatTimeoutInt, PrevExist: ""}); err != nil {
		return fmt.Errorf("First member dir Set() failed: %v", err.Error())
	}

	// create initial member status
	memberJSON, err := c.generateMemberJSON()
	if err != nil {
		return fmt.Errorf("Unable to generate initial member JSON: %v", err.Error())
	}

	if err := c.DalClient.Set(memberDir+"/status", memberJSON, nil); err != nil {
		return fmt.Errorf("Unable to create initial state: %v", err.Error())
	}

	// create member config dir
	if err := c.DalClient.Set(memberDir+"/config", "", &dal.SetOptions{Dir: true, TTLSec: 0, PrevExist: ""}); err != nil {
		return fmt.Errorf("Creating member config dir failed: %v", err.Error())
	}

	return nil
}

// ALWAYS: send member heartbeat updates
func (c *Cluster) runMemberHeartbeat() {
	llog := c.Log.WithField("method", "runMemberHeartbeat")

	llog.Debug("Launching member heartbeat...")

	memberDir := fmt.Sprintf("cluster/members/%v", c.MemberID)
	heartbeatTimeoutInt := int(time.Duration(c.Config.HeartbeatTimeout).Seconds())

	// create initial member dir
	if err := c.createInitialMemberStructure(memberDir, heartbeatTimeoutInt); err != nil {
		logFatal(llog, log.Fields{"err": err}, "Unable to create initial member dir")
		// llog.WithField("err", err).Fatal("Unable to create initial member dir")
	}

	// Avoid data structure creation/existence race
	c.initFinished <- true

	c.MemberHeartbeatLooper.Loop(func() error {
		// Unlikely error, but let's check jic
		memberJSON, err := c.generateMemberJSON()
		if err != nil {
			c.Config.EQClient.AddWithErrorLog("error",
				fmt.Sprintf("Unable to generate member JSON (retrying in %v)", c.Config.HeartbeatInterval.String()),
				llog, log.Fields{"err": err})

			return nil
		}

		// set status key (could fail)
		if err := c.DalClient.Set(
			memberDir+"/status", memberJSON,
			&dal.SetOptions{
				Dir:           false,
				TTLSec:        0,
				PrevExist:     "",
				CreateParents: true,
				Depth:         1,
			},
		); err != nil {
			c.Config.EQClient.AddWithErrorLog("error",
				fmt.Sprintf("Unable to save member JSON status (retrying in %v)", c.Config.HeartbeatInterval.String()),
				llog, log.Fields{"err": err})

			// Let's tell overwatch that something bad happened with backend
			c.OverwatchChan <- &overwatch.Message{
				Error:     fmt.Errorf("Unable to save key to etcd: %v", err),
				Source:    fmt.Sprintf("%v.runMemberHeartbeat", c.Identifier),
				ErrorType: overwatch.ETCD_GENERIC_ERROR,
			}

			return nil
		}

		// refresh dir
		go func(memberDir string, ttl int) {
			if err := c.DalClient.Refresh(memberDir, heartbeatTimeoutInt); err != nil {
				// Not sure if this should cause a member dropout
				c.Config.EQClient.AddWithErrorLog("error",
					fmt.Sprintf("Unable to refresh member dir (retrying in %v)", c.Config.HeartbeatInterval.String()),
					llog, log.Fields{"dir": memberDir, "err": err})

				c.OverwatchChan <- &overwatch.Message{
					Error:     fmt.Errorf("Unable to refresh key in etcd: %v", err),
					Source:    fmt.Sprintf("%v.runMemberHeartbeat", c.Identifier),
					ErrorType: overwatch.ETCD_GENERIC_ERROR,
				}
			}
		}(memberDir, heartbeatTimeoutInt)

		return nil
	})

	llog.Debug("Exiting...")
}

func (c *Cluster) generateMemberJSON() (string, error) {
	memberJSON := &MemberJSON{
		MemberID:      c.MemberID,
		Hostname:      c.Hostname,
		ListenAddress: c.Config.ListenAddress,
		LastUpdated:   time.Now(),
		Tags:          c.Config.Tags,
		Version:       c.Config.Version,
		SemVer:        c.Config.SemVer,
	}

	data, err := json.Marshal(memberJSON)
	if err != nil {
		return "", fmt.Errorf("Unable to marshal memberJSON: %v", err.Error())
	}

	return string(data), nil
}

func (c *Cluster) getState() (*DirectorJSON, error) {
	// Fetch the current state
	data, err := c.DalClient.Get(DIRECTOR_KEY, nil)

	if c.DalClient.IsKeyNotFound(err) {
		c.Log.Debug("No active director found")
		return nil, nil
	}

	if err != nil {
		c.Log.WithField("err", err).Warning("Unexpected dal get error")
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
		c.Log.Info("No existing director entry found - changing state!")
		return c.changeState(START, nil, CREATE)
	}

	// etcd says we are director, but we do not realize it
	// (ie. someone updated etcd manually and set us as director)
	if directorJSON.MemberID == c.MemberID {
		if !c.amDirector() {
			c.Log.Info("Not a director, but etcd says we are (updating state)!")
			return c.changeState(START, directorJSON, UPDATE) // update so we can compareAndSwap
		}

		// This case is hit when we got started back up via overwatch (still a
		// director but need to let director know to take over checks once more)
		if c.restarted {
			c.restarted = false
			return c.changeState(START, nil, NOOP) // update so we can compareAndSwap
		}
	}

	// etcd says we are not director, but we think we are
	// (dealing with a potential race?)
	if directorJSON.MemberID != c.MemberID {
		if c.amDirector() {
			c.Log.Info("Running in director mode, but etcd says we are not!")
			return c.changeState(STOP, nil, NOOP)
		}
	}

	// happy path
	if directorJSON.MemberID != c.MemberID {
		if c.isExpired(directorJSON.LastUpdate) {
			c.Log.Infof("Current director '%v' expired; time to upscale!", directorJSON.MemberID)
			return c.changeState(START, directorJSON, UPDATE)
		} else {
			c.Log.Infof("Current director '%v' not expired yet; waiting patiently", directorJSON.MemberID)
		}
	}

	// Nothing happening
	return nil
}

func (c *Cluster) changeState(action int, prevDirectorJSON *DirectorJSON, etcdAction int) error {
	if action == START {
		c.Log.Info("Taking over director role")

		// Only attempt to update state if we have to write to etcd (for UPDATE/CREATE)
		if etcdAction != NOOP {
			if err := c.updateState(prevDirectorJSON, etcdAction); err != nil {
				return fmt.Errorf("Unable to update director state: %v", err.Error())
			}
		}

		// Notify things to start? (ie. DirectorHeartbeat)
		c.setDirectorState(true)
	} else {
		c.Log.Info("Shutting down director role")

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

	c.Log.Debugf("Successfully %vd director state in dal", actionVerb)

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

	return c.DirectorState
}
