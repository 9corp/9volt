package overwatch

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"

	d "github.com/relistan/go-director"

	"github.com/9corp/9volt/base"
	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/fakes/dalfakes"
)

var _ = Describe("overwatch", func() {
	var (
		o            *Overwatch
		fakeDAL      *dalfakes.FakeIDal
		cfg          *config.Config
		watchChannel chan *Message
		components   []base.IComponent
	)

	BeforeEach(func() {
		fakeDAL = &dalfakes.FakeIDal{}
		o = New(cfg, watchChannel, components)
	})

	Context("New", func() {
		It("returns a filled out overwatch instance", func() {

		})
	})

	Context("Start", func() {
		It("starts the listener", func() {

		})
	})

	Context("runListener", func() {
		Context("when it receives a message on the watch channel", func() {
			It("does nothing if already in watch mode", func() {

			})

			It("sets active watch to true and stops the world", func() {

			})
		})
	})

	Context("stopTheWorld", func() {
		It("attempts to stop all components", func() {

		})

		It("launches a goroutine to begin watching deps", func() {

		})

		It("updates the healthcheck state", func() {

		})
	})

	Context("handleWatch", func() {
		Context("on recognized (etcd) message error types", func() {
			It("begins etcd watches", func() {

			})
		})

		Context("on unsupported message error types", func() {
			It("returns an error", func() {

			})
		})
	})

	Context("startTheWorld", func() {
		Context("happy path", func() {
			It("will start all components", func() {

			})

			It("will update healthstate", func() {

			})
		})

		Context("encounters an error while starting a component", func() {
			It("wil record it in error list and return a combined error", func() {

			})

			It("will not update healthstate", func() {

			})
		})
	})

	// TODO
	Context("beginEtcdWatch", func() {

	})
})
