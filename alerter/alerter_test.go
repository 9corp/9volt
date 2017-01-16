package alerter

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("alerter", func() {
	Context("New", func() {
		PIt("should return a new instance of Alerter")
	})

	Context("Start", func() {
		PIt("should create a map of alerters")
		PIt("should start the runner")
	})

	Context("run", func() {
		PIt("should receive messages on the message channel")
		PIt("should assign a uuid to an incoming message")
		PIt("should send alert message")
	})

	Context("handleMessage", func() {
		Context("happy path", func() {
			PIt("should be able to send alerts for multiple alerters without errors")
		})

		PIt("should return error if message fails validation")
		PIt("should log and append error to errorList if unable to load specific alerter config")
		PIt("should log and append error to errorList if unable to complete alerter specific validate")
		PIt("should log and append error to errorList if alerter message send fails")
		Pit("should add event and log error if errorList is not empty")
	})

	Context("loadAlerterConfig", func() {
		PIt("should fetch, load and return given alerter config pointer")
		PIt("should return error when dal fails to fetch alerter config")
		PIt("should return error when unable to unmarshal alerter config")
		PIt("should return error when unmarshaled alert config contains alerter type that we do not support")
	})

	Context("validateMessage", func() {
		PIt("should not return any errors with a valid alerter config")
		PIt("should error with a 0 length Key")
		PIt("should error if Source is not set")
		PIt("should error if Contents is nil")
		PIt("should error msg.Type does not contain a supported type")
	})
})
