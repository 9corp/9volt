package overwatch

import (
	"errors"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	d "github.com/relistan/go-director"

	"github.com/9corp/9volt/base"
	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/fakes/basefakes"
	"github.com/9corp/9volt/fakes/dalfakes"
)

var _ = Describe("overwatch", func() {
	var (
		o              *Overwatch
		fakeDAL        *dalfakes.FakeIDal
		cfg            *config.Config
		watchChannel   chan *Message
		fakeComponent  *basefakes.FakeIComponent
		components     []base.IComponent
		watchLooper    d.Looper
		listenerLooper d.Looper
	)

	BeforeEach(func() {
		fakeDAL = &dalfakes.FakeIDal{}
		fakeComponent = &basefakes.FakeIComponent{}
		fakeComponent.IdentifyReturns("fakeComponent")
		components = []base.IComponent{fakeComponent}
		listenerLooper = d.NewFreeLooper(d.ONCE, make(chan error, 1))
		watchLooper = d.NewFreeLooper(d.ONCE, make(chan error, 1))
		watchChannel = make(chan *Message, 1)

		cfg = &config.Config{
			DalClient: fakeDAL,
			Health: &config.Health{
				Lock: &sync.Mutex{},
			},
		}

		o = &Overwatch{
			Config:       cfg,
			WatchChannel: watchChannel,
			Components:   components,
			WatchLooper:  watchLooper,
			Looper:       listenerLooper,
			Component: base.Component{
				Identifier: "overwatch",
			},
		}
	})

	Context("New", func() {
		It("returns a filled out overwatch instance", func() {
			ow := New(cfg, watchChannel, components)

			Expect(ow).ToNot(BeNil())
			Expect(len(ow.Components)).To(Equal(len(components)))
			Expect(ow.Identifier).To(Equal("overwatch"))
		})
	})

	Context("Start", func() {
		It("starts the listener", func() {
			fakeDAL.NewWatcherForOverwatchReturns(nil, errors.New("foo"))

			o.Start()

			time.Sleep(100 * time.Millisecond)

			// Verify that runListener got executed
			watchChannel <- &Message{ErrorType: ETCD_GENERIC_ERROR}

			time.Sleep(100 * time.Millisecond)
			Expect(fakeComponent.StopCallCount()).To(Equal(1))
		})
	})

	Context("runListener", func() {
		Context("when a message is received on the watch channel", func() {
			BeforeEach(func() {
				watchChannel <- &Message{
					Source:    "foo",
					Error:     errors.New("etcd error"),
					ErrorType: ETCD_WATCHER_ERROR,
				}
			})

			It("does nothing if already in watch mode", func() {
				o.activeWatch = true

				o.runListener()

				// stopTheWorld() should not have been called
				Expect(fakeComponent.StopCallCount()).To(Equal(0))
			})

			It("sets active watch to true and stops the world", func() {
				o.activeWatch = false
				o.Config.Health.Ok = true

				// watch will fail, but we do not care about this in this test case
				fakeDAL.NewWatcherForOverwatchReturns(nil, errors.New("foo"))

				o.runListener()
				err := o.Looper.Wait()

				Expect(err).ToNot(HaveOccurred())

				// Another hack, waiting for goroutine to run
				time.Sleep(100 * time.Millisecond)
				Expect(fakeComponent.StopCallCount()).To(Equal(1))
				Expect(fakeComponent.IdentifyCallCount()).To(Equal(1))
				Expect(o.Config.Health.Ok).To(BeFalse())
			})
		})
	})

	Context("stopTheWorld", func() {
		BeforeEach(func() {
			fakeDAL.NewWatcherForOverwatchReturns(nil, errors.New("foo"))
		})

		It("attempts to stop all components", func() {
			o.stopTheWorld(&Message{ErrorType: ETCD_GENERIC_ERROR})
			time.Sleep(100 * time.Millisecond)

			Expect(fakeComponent.StopCallCount()).To(Equal(1))
		})

		It("launches a goroutine to begin watching deps", func() {
			o.stopTheWorld(&Message{ErrorType: ETCD_GENERIC_ERROR})
			time.Sleep(100 * time.Millisecond)

			Expect(fakeDAL.NewWatcherForOverwatchCallCount()).To(Equal(1))
		})

		It("updates the healthcheck state", func() {
			o.Config.Health.Ok = true

			o.stopTheWorld(&Message{ErrorType: ETCD_GENERIC_ERROR})
			time.Sleep(100 * time.Millisecond)

			Expect(o.Config.Health.Ok).To(BeFalse())
		})
	})

	Context("handleWatch", func() {
		Context("on recognized (etcd) message error types", func() {
			It("begins etcd watches", func() {
				fakeDAL.NewWatcherForOverwatchReturns(nil, errors.New("foo"))
				errorTypes := []int{ETCD_WATCHER_ERROR, ETCD_GENERIC_ERROR}

				for _, errorType := range errorTypes {
					o.handleWatch(&Message{ErrorType: errorType})
					time.Sleep(100 * time.Millisecond)
				}

				Expect(fakeDAL.NewWatcherForOverwatchCallCount()).To(Equal(len(errorTypes)))
			})
		})

		Context("on unsupported message error types", func() {
			It("returns an error", func() {
				err := o.handleWatch(&Message{ErrorType: 12345})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unknown error type"))
			})
		})
	})

	Context("startTheWorld", func() {
		Context("happy path", func() {
			It("will start all components", func() {
				err := o.startTheWorld()

				Expect(err).ToNot(HaveOccurred())
				Expect(fakeComponent.StartCallCount()).To(Equal(1))
				Expect(fakeComponent.IdentifyCallCount()).To(Equal(1))
			})

			It("will update healthstate", func() {
				o.Config.Health.Ok = false
				err := o.startTheWorld()

				Expect(o.Config.Health.Ok).To(BeTrue())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("encounters an error while starting a component", func() {
			It("wil record it in error list and return a combined error", func() {
				fakeComponent.StartReturns(errors.New("foo"))

				err := o.startTheWorld()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Ran into one or more errors"))
			})

			It("will not update healthstate", func() {
				o.Config.Health.Ok = false
				fakeComponent.StartReturns(errors.New("foo"))

				o.startTheWorld()

				Expect(o.Config.Health.Ok).To(BeFalse())
			})
		})
	})

	// TODO
	Context("Stop", func() {
		It("returns nil", func() {
			Expect(o.Stop()).To(BeNil())
		})
	})

	// TODO
	Context("beginEtcdWatch", func() {

	})
})
