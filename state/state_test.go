package state

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("state", func() {
	Context("New", func() {
		PIt("should return an intstance of state")
	})

	Context("Start", func() {
		PIt("should start reader and dumper")
	})

	Context("runReader", func() {
		PIt("should receive state message and save it to data map")
	})

	Context("runDumper", func() {
		PIt("dump contents from data map via dal + delete event from data map")
		PIt("return nil if data map is empty")
		PIt("log error if unable to marshal state event + do not delete event from data map")
		PIt("log error if unable to save state via dal + do not delete event from data map")
	})
})
