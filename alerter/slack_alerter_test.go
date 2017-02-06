package alerter

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("slack_alerter", func() {
	Context("NewSlack", func() {
		PIt("should return an instance of Slack")
	})

	Context("Send", func() {
		PIt("should send an alert message given a good message + alerter config")
		PIt("should return error if slack client returns error")
	})

	Context("Identify", func() {
		PIt("should return identifier string")
	})

	Context("ValidateConfig", func() {
		PIt("should return nil if given alerter config is properly filled out")
		PIt("should return error if options are not set")
		PIt("should return error if options are missing a required field")
	})

	Context("generateParams", func() {
		By("having custom username and icon-url alertConfig settings")
		PIt("should return slack message parameters containing custom settings")

		PIt("should return slack params with messageColor set to red if type is critical")
		PIt("should return slack params with messageColor set to green if type is resolve")
		PIt("should return slack params with messageColor set to orange if type is warning")
		PIt("should return slack params with correct fields")
	})
})
