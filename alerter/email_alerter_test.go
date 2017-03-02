package alerter

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("email_alerter", func() {
	Context("NewEmail", func() {
		PIt("should return an instance of Email")
	})

	Context("Send", func() {
		PIt("should send an email alert with a filled out alerter config + message")
		PIt("should set from field if given a custom from option")
		PIt("should set authentication options if auth is filled out in alerter config")
		PIt("should return an error if message send fails")
	})

	Context("Identify", func() {
		PIt("should return identifier string")
	})

	Context("ValidateConfig", func() {
		PIt("should return nil if given alerter config is properly filled out")
		PIt("should return error if options are not set")

		By("having 'auth' set in the alert config options")
		PIt("should return error if auth is set to something other than 'plain' or 'md5'")

		By("having username set")
		PIt("should error if password is not set")

		PIt("should return joined errors if more than one error is detected")
	})

	Context("generateMessage", func() {
		PIt("should return a message containing to, subject, detailed message and error details")
	})

	Context("generateAuth", func() {
		PIt("should return nil, nil if options do not contain username or password")
		PIt("should return error if auth is set to something other than plain or md5")
		PIt("should return smtp.Auth with 'plain' auth if 'auth' is not set in alerter config")
		PIt("should return error if 'address' is not configured in alerter config")
		PIt("should return error if 'address' is not in the correct host:port format")
	})
})
