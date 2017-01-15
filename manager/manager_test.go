package manager

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("manager", func() {
	Context("New", func() {
		PIt("return an instance of Manager")
	})

	Context("Start", func() {
		PIt("Start the manager run")
	})

	Context("run", func() {
		PIt("should start a monitor on a 'set' action")
		PIt("should stop a monitor on a 'delete' action")
		PIt("should not do anything on an ignorable action")
		PIt("should error on encountering a watcher error")
		PIt("should error on a 'unknown' action")
	})

	Context("ignorableWatcherEvent", func() {
		PIt("should return true if resp is nil")
		PIt("should return true if resp key is ignorable")
		PIt("should return false if resp key is not ignorable")
	})
})
