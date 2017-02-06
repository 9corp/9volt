package monitor

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("http_monitor", func() {
	Context("NewHTTPMonitor", func() {
		// Not very easy to test monitor internal settings due to constructor returning an interface
		PIt("should return IMonitor instance")
	})

	Context("Validate", func() {
		PIt("should return nil with correct settings")
		PIt("should return error if timeout exceeds or is equal to ")
	})

	Context("httpCheck", func() {
	})

	Context("constructURL", func() {
	})

	Context("performRequest", func() {
	})
})
