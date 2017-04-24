package cluster

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	d "github.com/relistan/go-director"

	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/fakes/dalfakes"
	"github.com/9corp/9volt/fakes/eventfakes"
	"github.com/9corp/9volt/overwatch"
	"github.com/9corp/9volt/util"
)

var _ = Describe("cluster", func() {
	var (
		c                   *Cluster
		fakeDAL             *dalfakes.FakeIDal
		fakeEventClient     *eventfakes.FakeIClient
		looperChan          chan error
		stateChan           chan bool
		distributeChan      chan bool
		overwatchChan       chan *overwatch.Message
		memberHeartbeatChan chan error
		testBuffer          *bytes.Buffer
		logger              *log.Logger

		directorID string
	)

	BeforeEach(func() {
		fakeDAL = &dalfakes.FakeIDal{}
		fakeEventClient = &eventfakes.FakeIClient{}
		looperChan = make(chan error, 1)
		stateChan = make(chan bool, 1)
		distributeChan = make(chan bool, 1)
		memberHeartbeatChan = make(chan error, 1)
		overwatchChan = make(chan *overwatch.Message, 1)
		logger = log.New()
		logger.Out = bufio.NewWriter(testBuffer)

		directorID = "myid123"

		c = &Cluster{
			DalClient:     fakeDAL,
			DirectorLock:  &sync.Mutex{},
			Log:           logger,
			MemberID:      directorID,
			Hostname:      "hostname",
			DirectorState: true,
			StateChan:     stateChan,
			OverwatchChan: overwatchChan,
			Config: &config.Config{
				EtcdMembers:  []string{"1", "2", "3"},
				EtcdPrefix:   "9volt",
				EtcdUserPass: "user:pass",
				EQClient:     fakeEventClient,
			},
			initFinished: make(chan bool, 1),
		}
	})

	Context("New", func() {
		// TODO
		PIt("should return a cluster instance")

		It("should error if dal instantiation fails", func() {
			c.Config.EtcdUserPass = "invalid"
			newCluster, err := New(c.Config, stateChan, distributeChan, overwatchChan)

			Expect(err).To(HaveOccurred())
			Expect(newCluster).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("Bad username/password passed"))
		})

		// TODO
		PIt("should error if hostname fetching fails")
	})

	Context("Start", func() {
		PIt("should start all components")
		PIt("should wait until initFinished message received before start runMemberMonitor")
	})

	Context("runDirectorMonitor", func() {
		BeforeEach(func() {
			c.DirectorMonitorLooper = d.NewFreeLooper(d.ONCE, make(chan error))
		})

		JustBeforeEach(func() {
			go c.runDirectorMonitor()
		})

		Context("happy path", func() {
			BeforeEach(func() {
				By("is the director already")
				fakeDAL.GetReturns(
					map[string]string{
						DIRECTOR_KEY: fmt.Sprintf(`{"MemberID":"%s","LastUpdate":"2017-03-05T09:39:54.465896214-08:00"}`, directorID),
					},
					nil,
				)
			})

			It("should do nothing", func() {
				err := c.DirectorMonitorLooper.Wait()
				Expect(err).ToNot(HaveOccurred())
				Expect(c.DirectorState).To(BeTrue())
			})

		})

		Context("get state fails", func() {
			BeforeEach(func() {
				fakeDAL.GetReturns(nil, errors.New("some error"))
			})

			It("should log an error event", func() {
				err := c.DirectorMonitorLooper.Wait()
				Expect(err).ToNot(HaveOccurred())

				key, msg, _, _ := fakeEventClient.AddWithErrorLogArgsForCall(0)
				Expect(key).To(Equal("error"))
				Expect(msg).To(ContainSubstring("Unable to fetch director state"))
				// Expect(msg).To(ContainSubstring("some error"))
			})
		})

		Context("handle state fails", func() {
			BeforeEach(func() {
				By("not director but etcd disagrees")
				c.DirectorState = false
				fakeDAL.GetReturns(
					map[string]string{
						DIRECTOR_KEY: fmt.Sprintf(`{"MemberID":"%s","LastUpdate":"2017-03-05T09:39:54.465896214-08:00"}`, directorID),
					},
					nil,
				)
				fakeDAL.UpdateDirectorStateReturns(errors.New("failed that"))
			})

			It("should log an error event", func() {
				err := c.DirectorMonitorLooper.Wait()
				Expect(err).ToNot(HaveOccurred())

				key, msg, _, _ := fakeEventClient.AddWithErrorLogArgsForCall(0)
				Expect(key).To(Equal("error"))
				Expect(msg).To(ContainSubstring("Unable to handle state"))
				// Expect(msg).To(ContainSubstring("failed that"))
			})
		})
	})

	Context("runDirectorHeartbeat", func() {
		BeforeEach(func() {
			c.DirectorHeartbeatLooper = d.NewFreeLooper(d.ONCE, looperChan)

		})

		JustBeforeEach(func() {
			c.runDirectorHeartbeat()
		})

		Context("happy path", func() {
			BeforeEach(func() {
				fakeDAL.UpdateDirectorStateReturns(nil)
			})

			It("should send director heartbeat", func() {
				err := <-looperChan
				Expect(err).To(BeNil())
				Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(1))
			})
		})

		Context("when not director", func() {
			BeforeEach(func() {
				c.DirectorState = false
			})

			It("should not do anything if not director", func() {
				Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(0))
			})
		})

		Context("when heartbeat send fails", func() {
			BeforeEach(func() {
				fakeDAL.UpdateDirectorStateReturns(errors.New("generic error"))
			})

			It("should add event log and send message to overwatch", func() {
				key, msg, _, _ := fakeEventClient.AddWithErrorLogArgsForCall(0)

				Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(1))
				Expect(key).To(Equal("error"))
				Expect(msg).To(ContainSubstring("Unable to send director heartbeat"))

				time.Sleep(100 * time.Millisecond)
				overwatchMsg := <-overwatchChan

				Expect(overwatchMsg.Error.Error()).To(ContainSubstring("Potential etcd write error"))
			})
		})
	})

	Context("sendDirectorHeartbeat", func() {
		var (
			err error
		)

		JustBeforeEach(func() {
			err = c.sendDirectorHeartbeat()
		})

		Context("happy path", func() {
			It("should update director state via dal", func() {
				Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(1))
			})

			It("should generate valid json", func() {
				data, prevValue, force := fakeDAL.UpdateDirectorStateArgsForCall(0)

				Expect(err).To(BeNil())
				Expect(prevValue).To(Equal(""))
				Expect(force).To(BeTrue())

				var directorJSON DirectorJSON

				Expect(json.Unmarshal([]byte(data), &directorJSON)).ToNot(HaveOccurred())
				Expect(directorJSON.MemberID).To(Equal(c.MemberID))
			})
		})

		Context("if director state fails to update", func() {
			BeforeEach(func() {
				fakeDAL.UpdateDirectorStateReturns(errors.New("foo"))
			})

			It("should return error", func() {
				Expect(err).To(Equal(errors.New("Unable to update director heartbeat: foo")))
			})
		})
	})

	Context("runMemberMonitor", func() {
		PIt("should perform check distribution on 'set' and 'expire' actions")
		PIt("should not do anything if not director")
		PIt("should ignore watcher event if key is dir or contains 'config'")
		PIt("should add event and log error if watcher returns an error")
		PIt("should do nothing on unrecognized event actions")
	})

	Context("createInitialMemberStructure", func() {
		var (
			memberDir = "123456"
		)

		BeforeEach(func() {
			c.Config = &config.Config{
				ListenAddress: "127.0.0.1:8080",
				Tags:          []string{"1", "2", "3"},
				Version:       "asdf",
				SemVer:        "0.0.1",
			}
		})

		Context("happy path", func() {
			It("should delete member dir if member dir exists", func() {
				fakeDAL.KeyExistsReturns(true, true, nil)

				err := c.createInitialMemberStructure(memberDir, 1)

				Expect(err).ToNot(HaveOccurred())
				Expect(fakeDAL.KeyExistsCallCount()).To(Equal(1))
				Expect(fakeDAL.KeyExistsArgsForCall(0)).To(Equal(memberDir))
				Expect(fakeDAL.DeleteCallCount()).To(Equal(1))

				argKey, argRecursive := fakeDAL.DeleteArgsForCall(0)

				Expect(argKey).To(Equal(memberDir))
				Expect(argRecursive).To(Equal(true))
			})

			It("should not delete memberDir if memberDir does not exist", func() {
				fakeDAL.KeyExistsReturns(false, true, nil)

				err := c.createInitialMemberStructure(memberDir, 1)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeDAL.DeleteCallCount()).To(Equal(0))
			})

			It("should create new member dir", func() {
				fakeDAL.KeyExistsReturns(true, true, nil)

				err := c.createInitialMemberStructure(memberDir, 1)

				argDir, _, argOptions := fakeDAL.SetArgsForCall(0)

				Expect(argDir).To(Equal(memberDir))
				Expect(argOptions.Dir).To(Equal(true))
				Expect(err).ToNot(HaveOccurred())
			})

			It("should create initial member status", func() {
				fakeDAL.KeyExistsReturns(true, true, nil)
				err := c.createInitialMemberStructure(memberDir, 1)
				Expect(err).ToNot(HaveOccurred())

				argKey, argVal, argOptions := fakeDAL.SetArgsForCall(1)
				Expect(argKey).To(Equal(memberDir + "/status"))

				// Verify we pushed valid json
				var memberJSON MemberJSON
				Expect(json.Unmarshal([]byte(argVal), &memberJSON)).ToNot(HaveOccurred())
				Expect(memberJSON.Hostname).To(Equal(c.Hostname))
				Expect(memberJSON.MemberID).To(Equal(directorID))

				Expect(argOptions).To(BeNil())
			})

			It("should create member config dir", func() {
				fakeDAL.KeyExistsReturns(true, true, nil)
				err := c.createInitialMemberStructure(memberDir, 1)
				Expect(err).ToNot(HaveOccurred())

				argKey, argVal, argOptions := fakeDAL.SetArgsForCall(2)

				Expect(argKey).To(Equal(memberDir + "/config"))
				Expect(argVal).To(Equal(""))
				Expect(argOptions.Dir).To(BeTrue())
				Expect(argOptions.TTLSec).To(Equal(0))
			})
		})

		It("should error if dal fails to perform member existence check", func() {
			fakeDAL.KeyExistsReturns(true, true, errors.New("foo"))
			err := c.createInitialMemberStructure(memberDir, 1)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to verify pre-existence of member dir: foo"))
		})

		It("should error if delete fails if memberdir exists", func() {
			fakeDAL.KeyExistsReturns(true, true, nil)
			fakeDAL.DeleteReturns(errors.New("foo"))
			err := c.createInitialMemberStructure(memberDir, 1)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to delete pre-existing member dir"))
		})

		It("should error if dal fails to create member dir", func() {
			fakeDAL.KeyExistsReturns(true, true, nil)
			fakeDAL.SetReturns(errors.New("foo"))

			err := c.createInitialMemberStructure(memberDir, 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("First member dir Set() failed: foo"))
		})

		// The only potential failure here is due to JSON marshaling - difficult to test
		PIt("should error if unable to generate initial member status")

		It("should error if dal fails to save initial member status", func() {
			fakeDAL.KeyExistsReturns(true, true, nil)
			fakeDAL.SetStub = func(key, val string, options *dal.SetOptions) error {
				if key == memberDir+"/status" {
					return errors.New("foo")
				}

				return nil
			}

			err := c.createInitialMemberStructure(memberDir, 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Unable to create initial state: foo"))
		})

		It("should error if dal fails to create initial member config dir", func() {
			fakeDAL.KeyExistsReturns(true, true, nil)
			fakeDAL.SetStub = func(key, val string, options *dal.SetOptions) error {
				if key == memberDir+"/config" {
					return errors.New("foo")
				}

				return nil
			}

			err := c.createInitialMemberStructure(memberDir, 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Creating member config dir failed: foo"))
		})
	})

	Context("runMemberHeartbeat", func() {
		BeforeEach(func() {
			c.Config.HeartbeatTimeout = util.CustomDuration(time.Duration(10) * time.Second)
			c.MemberHeartbeatLooper = d.NewFreeLooper(d.ONCE, memberHeartbeatChan)
		})

		Context("happy path", func() {
			It("should create initial member structure (call right number of Set()'s)", func() {
				c.runMemberHeartbeat()
				<-c.initFinished

				// createInitialMemberStructure was called
				Expect(fakeDAL.KeyExistsCallCount()).To(Equal(1))
				Expect(fakeDAL.SetCallCount()).To(Equal(4)) // + 1 for .Set within runMemberHeartbeat
			})

			It("should send an initFinished notification", func() {
				c.runMemberHeartbeat()
				called := <-c.initFinished
				Expect(called).To(BeTrue())
			})

			It("should set member status in etcd", func() {
				c.runMemberHeartbeat()
				<-c.initFinished

				key, value, options := fakeDAL.SetArgsForCall(3)
				var memberJSON MemberJSON

				Expect(key).To(Equal("cluster/members/" + directorID + "/status"))
				Expect(options.Dir).To(BeFalse())
				Expect(options.CreateParents).To(BeTrue())
				Expect(options.Depth).To(Equal(1))
				Expect(json.Unmarshal([]byte(value), &memberJSON)).ToNot(HaveOccurred())
				Expect(memberJSON.MemberID).To(Equal(directorID))
				Expect(memberJSON.Hostname).To(Equal(c.Hostname))
			})

			It("should refresh its own member dir in etcd", func() {
				c.runMemberHeartbeat()
				<-c.initFinished

				// hacky, but eh -- need to make sure the goroutine has had time to run
				time.Sleep(500 * time.Millisecond)
				memberDir, heartbeatTimeout := fakeDAL.RefreshArgsForCall(0)

				Expect(fakeDAL.RefreshCallCount()).To(Equal(1))
				Expect(memberDir).To(Equal("cluster/members/" + directorID))
				Expect(heartbeatTimeout).To(Equal(int(time.Duration(c.Config.HeartbeatTimeout).Seconds())))
			})
		})

		Context("when unable to create initial member structure", func() {
			var (
				logFatalNumCalled int
				logFatalMsg       string
			)
			It("should exit", func() {
				logFatal = func(logger log.FieldLogger, fields log.Fields, msg string) {
					logFatalNumCalled++
					logFatalMsg = msg
				}

				// cause createInitialMemberStructure to fail
				fakeDAL.KeyExistsReturns(false, false, errors.New("foo"))

				c.runMemberHeartbeat()

				Expect(logFatalNumCalled).To(Equal(1))
				Expect(logFatalMsg).To(ContainSubstring("Unable to create initial member dir"))
			})
		})

		// Not sure of an easy way to get this path tested
		Context("when unable to generate member json", func() {
			PIt("should add event land log error")
		})

		Context("when unable to save memeber status", func() {
			var (
				iter int
			)

			It("should add event log with failure and send message to overwatch", func() {
				fakeDAL.SetStub = func(key, val string, options *dal.SetOptions) error {
					// On the third Set() call, cause it to fail
					defer func() { iter++ }()
					if iter == 3 {
						return errors.New("foo")
					}

					return nil
				}

				c.runMemberHeartbeat()
				<-c.initFinished
				overwatchMsg := <-overwatchChan

				Expect(fakeDAL.SetCallCount()).To(Equal(4))
				Expect(fakeEventClient.AddWithErrorLogCallCount()).To(Equal(1))
				Expect(overwatchMsg.Error.Error()).To(ContainSubstring("Unable to save key to etcd:"))
			})
		})

		Context("when dal fails to refresh member dir", func() {
			It("should add event log with failure and send message to overwatch", func() {
				fakeDAL.RefreshReturns(errors.New("foo"))

				c.runMemberHeartbeat()
				<-c.initFinished
				overwatchMsg := <-overwatchChan

				Expect(fakeDAL.RefreshCallCount()).To(Equal(1))
				Expect(overwatchMsg.Error.Error()).To(ContainSubstring("Unable to refresh key in etcd"))
			})
		})
	})

	Context("generateMemberJSON", func() {
		BeforeEach(func() {
			c.Config = &config.Config{
				ListenAddress: "127.0.0.1:8080",
				Tags:          []string{"1", "2", "3"},
				Version:       "asdf",
				SemVer:        "0.0.1",
			}
		})

		It("should return a valid member state JSON blob", func() {
			data, err := c.generateMemberJSON()

			Expect(err).ToNot(HaveOccurred())

			var memberJSON MemberJSON

			Expect(json.Unmarshal([]byte(data), &memberJSON)).ToNot(HaveOccurred())
			Expect(memberJSON.Hostname).To(Equal(c.Hostname))
			Expect(memberJSON.Tags).To(Equal(c.Config.Tags))
			Expect(memberJSON.ListenAddress).To(Equal(c.Config.ListenAddress))
			Expect(memberJSON.SemVer).To(Equal(c.Config.SemVer))
		})

		// Not sure how this can be tested exactly, as json.Marshal is tested by
		// passing it a []interface{} with some math.* values.
		//
		// https://golang.org/src/encoding/json/encode_test.go
		PIt("should return error if unable to marshal member state struct")
	})

	Context("getState", func() {
		var (
			directorJSON *DirectorJSON
			err          error
		)

		Context("happy path", func() {
			By("having no existing director state")
			It("should return nil *DirectorJSON and no error", func() {
				fakeDAL.GetReturns(nil, errors.New("foo"))
				fakeDAL.IsKeyNotFoundReturns(true)
				directorJSON, err = c.getState()

				Expect(directorJSON).To(BeNil())
				Expect(err).To(BeNil())
			})

			By("having existing director state")
			It("should return a pointer to DirectorJSON and no error", func() {
				fakeDAL.GetReturns(map[string]string{
					DIRECTOR_KEY: "{\"MemberID\":\"" + directorID + "\",\"LastUpdate\":\"2017-04-16T16:54:54.262695405-07:00\"}",
				}, nil)
				fakeDAL.IsKeyNotFoundReturns(false)

				directorJSON, err = c.getState()

				Expect(err).To(BeNil())
				Expect(directorJSON).ToNot(BeNil())
				Expect(directorJSON.MemberID).To(Equal(directorID))
			})
		})

		Context("when dal fails to fetch state", func() {
			It("should return error", func() {
				fakeDAL.GetReturns(nil, errors.New("foo"))

				directorJSON, err = c.getState()
				Expect(err).To(Equal(errors.New("foo")))
				Expect(directorJSON).To(BeNil())
			})
		})

		Context("when returned state does not contain director key", func() {
			It("should return error", func() {
				fakeDAL.GetReturns(map[string]string{"foo": "bar"}, nil)

				directorJSON, err = c.getState()
				Expect(err.Error()).To(ContainSubstring("Uhh, no 'director'"))
				Expect(directorJSON).To(BeNil())
			})
		})

		Context("when unmarshalling director state json blob fails", func() {
			It("should return error", func() {
				fakeDAL.GetReturns(map[string]string{DIRECTOR_KEY: "invalid_json"}, nil)

				directorJSON, err = c.getState()
				Expect(err.Error()).To(ContainSubstring("Unable to unmarshal director"))
				Expect(directorJSON).To(BeNil())
			})
		})
	})

	Context("handleState", func() {
		var (
			givenDirector *DirectorJSON
			err           error
		)

		BeforeEach(func() {
			givenDirector = &DirectorJSON{
				LastUpdate: time.Now(),
				MemberID:   "diff-director-id",
			}
		})

		Context("when director json is nil", func() {
			It("should become director (and update state locally and in etcd  (via CREATE)", func() {
				c.DirectorState = false
				err = c.handleState(nil)

				Expect(err).ToNot(HaveOccurred())
				Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(1))
				Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(0))
				Expect(c.DirectorState).To(BeTrue())
			})
		})

		Context("when given director id matches our director id", func() {
			Context("and we are not already the director", func() {
				It("should become director (and update state locally and in etcd (via UPDATE)", func() {
					c.DirectorState = false
					givenDirector.MemberID = directorID

					err = c.handleState(givenDirector)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(0))
					Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(1))
					Expect(c.DirectorState).To(BeTrue())
				})
			})

			Context("when our component was previously stopped", func() {
				It("should toggle the director state to true (and not update etcd)", func() {
					c.DirectorState = true
					givenDirector.MemberID = directorID
					c.restarted = true

					err = c.handleState(givenDirector)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(0))
					Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(0))
					Expect(c.DirectorState).To(BeTrue())
					Expect(c.restarted).To(BeFalse())
				})
			})
		})

		Context("when given director id does not match our director id", func() {
			BeforeEach(func() {
				c.Config.HeartbeatTimeout = util.CustomDuration(time.Duration(10) * time.Second)
			})

			Context("but we *think* we are the director", func() {
				It("should stop being the director (and NOT update etcd)", func() {
					c.DirectorState = true

					err = c.handleState(givenDirector)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(0))
					Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(0))
					Expect(c.DirectorState).To(BeFalse())
				})
			})

			Context("and given director has expired", func() {
				It("should update local state and in etcd", func() {
					givenDirector.LastUpdate = time.Now().Add(time.Duration(-10) * time.Hour)
					c.DirectorState = false

					err = c.handleState(givenDirector)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(0))
					Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(1))
					Expect(c.DirectorState).To(BeTrue())
				})
			})

			Context("and given director has not expired", func() {
				It("should do nothing", func() {
					// no etcd update, no local state change
					givenDirector.LastUpdate = time.Now()
					c.DirectorState = false

					err = c.handleState(givenDirector)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(0))
					Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(0))
					Expect(c.DirectorState).To(BeFalse())
				})
			})
		})
	})

	Context("changeState", func() {
		var (
			prevDirector = &DirectorJSON{
				LastUpdate: time.Now(),
				MemberID:   "old-director-id",
			}

			err error
		)

		Context("happy path", func() {
			Context("when action is set to START", func() {
				Context("and etcdAction set to CREATE or UPDATE", func() {
					It("should update etcd", func() {
						fakeDAL.CreateDirectorStateReturns(nil)
						err = c.changeState(START, prevDirector, CREATE)

						Expect(err).ToNot(HaveOccurred())
						Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(1))
						Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(0))
					})
				})

				It("should update director state to true", func() {
					c.DirectorState = false
					err = c.changeState(START, prevDirector, UPDATE)
					Expect(err).ToNot(HaveOccurred())
					Expect(c.DirectorState).To(Equal(true))
				})
			})

			Context("when action is set to anything else", func() {
				It("should update director state (and NOT etcd)", func() {
					c.DirectorState = true
					err = c.changeState(STOP, prevDirector, CREATE)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(0))
					Expect(c.DirectorState).To(Equal(false))
				})
			})
		})

		Context("when update state fails", func() {
			It("should return error", func() {
				fakeDAL.CreateDirectorStateReturns(errors.New("foo"))

				err = c.changeState(START, prevDirector, CREATE)

				Expect(err.Error()).To(ContainSubstring("Unable to update director state"))
			})
		})
	})

	Context("setDirectorState", func() {
		It("should update internal director state + send change via state chan", func() {
			c.setDirectorState(true)
			recv := <-stateChan

			Expect(c.DirectorState).To(BeTrue())
			Expect(recv).To(BeTrue())
		})
	})

	Context("updateState", func() {
		Context("happy path", func() {
			var (
				prevDirector = &DirectorJSON{
					LastUpdate: time.Now(),
					MemberID:   "old-director-id",
				}
			)

			BeforeEach(func() {
				fakeDAL.UpdateDirectorStateReturns(nil)
				fakeDAL.CreateDirectorStateReturns(nil)
			})

			By("having etcdAction set to 'UPDATE'")
			It("should update director state", func() {
				err := c.updateState(prevDirector, UPDATE)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(1))
				Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(0))
			})

			By("having etcdAction set to 'CREATE'")
			It("should create new director state", func() {
				err := c.updateState(prevDirector, CREATE)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeDAL.CreateDirectorStateCallCount()).To(Equal(1))
				Expect(fakeDAL.UpdateDirectorStateCallCount()).To(Equal(0))
			})
		})

		Context("failure cases", func() {
			var (
				err error
			)

			Context("when given an action other than 'CREATE' or 'UPDATE'", func() {
				It("should error", func() {
					err = c.updateState(nil, 1234)
					Expect(err.Error()).To(ContainSubstring("Unrecognized etcdAction"))

				})
			})

			// Not sure if there's an easy way to achieve this state
			PIt("should error when unable to marshal new director state")

			Context("when update (or create) via dal fails", func() {
				var (
					prevDirector = &DirectorJSON{
						LastUpdate: time.Now(),
						MemberID:   "old-director-id",
					}
				)

				It("should error", func() {
					fakeDAL.UpdateDirectorStateReturns(errors.New("foo"))

					err = c.updateState(prevDirector, UPDATE)
					Expect(err.Error()).To(ContainSubstring("Unable to update director state"))

					fakeDAL.CreateDirectorStateReturns(errors.New("foo"))
					err = c.updateState(nil, CREATE)
					Expect(err.Error()).To(ContainSubstring("Unable to create director state"))
				})
			})
		})
	})

	Context("isExpired", func() {
		BeforeEach(func() {
			c.Config.HeartbeatTimeout = util.CustomDuration(time.Duration(10) * time.Second)
		})

		It("should return true if given datetime is older than NOW+HeartbeatTimeout", func() {
			now := time.Now().Add(time.Duration(-10) * time.Hour)
			Expect(c.isExpired(now)).To(BeTrue())
		})

		It("should return false if given datetime is NOT older than Now()+HeartbeatTimeout", func() {
			now := time.Now()
			Expect(c.isExpired(now)).To(BeFalse())
		})
	})

	Context("amDirector", func() {
		It("should return correct director state", func() {
			c.DirectorState = true
			Expect(c.amDirector()).To(Equal(c.DirectorState))

			c.DirectorState = false
			Expect(c.amDirector()).To(Equal(c.DirectorState))
		})
	})
})
