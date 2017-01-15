package cluster

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("cluster", func() {
	Context("New", func() {
		PIt("should return a cluster instance")
		PIt("should error if dal instantiation fails")
		PIt("should error if hostname fetching fails")
	})

	Context("Start", func() {
		PIt("should start all components")
		PIt("should wait until initFinished message received before start runMemberMonitor")
	})

	Context("runDirectorMonitor", func() {
		PIt("should handle state change")
		PIt("should add event and log error if unable to fetch director state")
		PIt("should add event and log error if unable to handle state change")
	})

	Context("runDirectorHeartbeat", func() {
		PIt("should periodically send director heartbeat")
		PIt("should not do anything if not director")
		PIt("should add event event and log error if heartbeat send fails")
	})

	Context("sendDirectorHeartbeat", func() {
		PIt("should update director state via dal")
		PIt("should return error if director state update fails")
		PIt("should return error if director state marshal fails")
	})

	Context("runMemberMonitor", func() {
		PIt("should perform check distribution on 'set' and 'expire' actions")
		PIt("should not do anything if not director")
		PIt("should ignore watcher event if key is dir or contains 'config'")
		PIt("should add event and log error if watcher returns an error")
		PIt("should do nothing on unrecognized event actions")
	})

	Context("createInitialMemberStructure", func() {
		Context("happy path", func() {
			PIt("should delete member dir if member dir exists")
			PIt("should create new member dir")
			PIt("should create initial member status")
			PIt("should create member config dir")
		})

		PIt("should error if dal fails to perform member existence check")
		PIt("should error if dal fails to create member dir")
		PIt("should error if unable to generate initial member status")
		PIt("should error if dal fails to save initial member status")
		Pit("should error if dal fails to create initial member config dir")
	})

	Context("runMemberHeartbeat", func() {
	})

	Context("generateMemberJSON", func() {
	})

	Context("getState", func() {
	})

	Context("handleState", func() {
	})

	Context("changeState", func() {
	})

	Context("setDirectorState", func() {
	})

	Context("updateState", func() {
	})

	Context("isExpired", func() {
	})

	Context("amDirector", func() {
	})
})
