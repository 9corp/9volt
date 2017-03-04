package monitor

import (
	"io/ioutil"
	"time"

	"github.com/9corp/9volt/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/jarcoal/httpmock.v1"
)

var _ = BeforeSuite(func() {
	// block all HTTP requests
	httpmock.Activate()
})

var _ = BeforeEach(func() {
	// remove any mocks
	httpmock.Reset()
})

var _ = AfterSuite(func() {
	httpmock.DeactivateAndReset()
})

var _ = Describe("http_monitor", func() {

	var (
		monitor *HTTPMonitor
		config  *RootMonitorConfig
		url     string
	)

	Context("NewHTTPMonitor", func() {
		It("should return IMonitor instance", func() {
			config = &RootMonitorConfig{
				Config: &MonitorConfig{
					HTTPURL:        "/health",
					Port:           31337,
					HTTPSSL:        true,
					Host:           "beowulf",
					HTTPStatusCode: 200,
				},
			}
			monitor = NewHTTPMonitor(config)

			Expect(monitor.Timeout).NotTo(Equal(util.CustomDuration(0)))
			Expect(monitor.MonitorFunc).NotTo(BeNil())
		})
	})

	Context("Validate", func() {
		BeforeEach(func() {
			config = &RootMonitorConfig{
				Config: &MonitorConfig{
					HTTPURL:        "/health",
					Port:           31337,
					HTTPSSL:        true,
					Host:           "beowulf",
					HTTPStatusCode: 200,
				},
			}
			monitor = NewHTTPMonitor(config)
		})

		It("should return nil with correct settings", func() {
			config.Config.Interval = util.CustomDuration(5 * time.Second)
			Expect(monitor.Validate()).To(BeNil())
		})

		It("should return error if timeout exceeds or is equal to interval", func() {
			err := monitor.Validate()
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("cannot equal or exceed"))
		})
	})

	Context("httpCheck", func() {
		BeforeEach(func() {
			config = &RootMonitorConfig{
				Config: &MonitorConfig{
					HTTPURL:        "/health",
					Port:           31337,
					HTTPSSL:        true,
					Host:           "beowulf",
					HTTPStatusCode: 200,
				},
			}
			monitor = NewHTTPMonitor(config)
			url = monitor.constructURL()
		})

		It("identifies failures in status code", func() {
			httpmock.RegisterResponder("GET", "https://beowulf:31337/health",
				httpmock.NewStringResponder(
					500, `{"status": "error", "message": "Bad Thing™!"}`,
				),
			)

			err := monitor.httpCheck()
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("does not match expected status"))
		})

		It("identifies success in status code", func() {
			httpmock.RegisterResponder("GET", "https://beowulf:31337/health",
				httpmock.NewStringResponder(
					200, `{"status": "error", "message": "Bad Thing™!"}`,
				),
			)

			err := monitor.httpCheck()
			Expect(err).To(BeNil())
		})

		It("identifies failure by missing sttring", func() {
			httpmock.RegisterResponder("GET", "https://beowulf:31337/health",
				httpmock.NewStringResponder(200, ""),
			)

			config.Config.Expect = "Amazing things!"

			err := monitor.httpCheck()
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("does not contain expected"))
		})
	})

	Context("constructURL", func() {
		BeforeEach(func() {
			config = &RootMonitorConfig{
				Config: &MonitorConfig{},
			}

			monitor = NewHTTPMonitor(config)
		})

		It("handles http URLs", func() {
			Expect(monitor.constructURL()).To(Equal("http:///"))
		})

		It("handles https URLs", func() {
			config.Config.HTTPSSL = true
			Expect(monitor.constructURL()).To(Equal("https:///"))
		})

		It("appends a port when specified", func() {
			config.Config.Port = 31337
			Expect(monitor.constructURL()).To(Equal("http://:31337/"))
		})

		It("does not duplicate leading slashes", func() {
			config.Config.HTTPURL = "/somewhere-out-there"
			config.Config.Port = 31337
			Expect(monitor.constructURL()).To(Equal("http://:31337/somewhere-out-there"))
		})

		It("guarantees a leading slash", func() {
			config.Config.HTTPURL = "a-place"
			Expect(monitor.constructURL()).To(Equal("http:///a-place"))
		})

		It("formats a complete URL", func() {
			config.Config.HTTPURL = "/health"
			config.Config.Port = 31337
			config.Config.HTTPSSL = true
			config.Config.Host = "beowulf"
			Expect(monitor.constructURL()).To(Equal("https://beowulf:31337/health"))
		})
	})

	Context("performRequest", func() {
		BeforeEach(func() {
			config = &RootMonitorConfig{
				Config: &MonitorConfig{
					HTTPURL: "/health",
					Port:    31337,
					HTTPSSL: true,
					Host:    "beowulf",
				},
			}
			monitor = NewHTTPMonitor(config)
			url = monitor.constructURL()
		})

		It("works with no body", func() {
			httpmock.RegisterResponder("GET", "https://beowulf:31337/health",
				httpmock.NewStringResponder(
					200, `{"status": "ok", "message": "good stuff!"}`,
				),
			)

			resp, err := monitor.performRequest("GET", url, "")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(200))
			_, err = ioutil.ReadAll(resp.Body)
			Expect(err).To(BeNil())
		})

		It("works with a body", func() {
			httpmock.RegisterResponder("GET", "https://beowulf:31337/health",
				httpmock.NewStringResponder(
					200, `{"status": "ok", "message": "good stuff!"}`,
				),
			)

			resp, err := monitor.performRequest("GET", url, "a request body")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(200))
		})

	})
})
