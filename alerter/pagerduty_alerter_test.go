package alerter

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("pagerduty_alerter", func() {
	Context("NewPagerduty", func() {
		PIt("should return an instance of Pagerduty")
	})

	Context("Send", func() {
		PIt("should create an event in pagerduty")
		PIt("should return an error if pagerduty returns an error")
	})

	Context("generateEvent", func() {
		PIt("should return an instance of a pagerduty event")
		By("having message type set to something other than 'resolve'")
		PIt("event type should be set to trigger")
		By("having message type set to resolve")
		PIt("event type should be set to resolve")
		// Not finished
	})
})
