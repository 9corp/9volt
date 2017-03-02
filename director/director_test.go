package director

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("director", func() {
	Context("New", func() {
		PIt("should return an instance of director")
		PIt("should error if dal instantiation fails")
	})

	Context("Start", func() {
		PIt("should start runDistributeListener")
		PIt("should start runStateListener")
		PIt("should start collectCheckStats")
	})

	Context("collectCheckStats", func() {
		PIt("should fetch stats via dal and save to CheckStats map")
		PIt("should add event entry + log if fetch check stats fails via dal")
	})

	Context("runDistributeListener", func() {
		PIt("should distribute checks upon receiving distribute message")
		PIt("should log a warning if we are not a director")
		PIt("should add event + log if distributeCheck fails")
	})

	Context("runStateListener", func() {
		PIt("should set director state")

		Context("on true state change", func() {
			PIt("should start config watcher")
			PIt("perform check distribution")
			PIt("should add event + log error when check distribution fails")
		})

		Context("on false state change", func() {
			PIt("should cancel the existing config watcher")
		})
	})

	Context("distributeChecks", func() {
		PIt("should performCheckDistribution and not error")
		PIt("should error if member existence check fails")
		PIt("should error if cluster member fetch fails")
		PIt("should error if cluster members length is 0")
		PIt("should error if cluster check key fetch fails")
		PIt("should error if check key length is 0")
		PIt("should error if performCheckDistribution fails")
	})

	Context("performCheckDistribution", func() {
		Context("happy path", func() {
			PIt("should clear old check references")
			PIt("should equally divide checks between members")
			PIt("should create new check references")
		})

		PIt("should error if dal fails on ClearCheckReferences")
		PIt("should error if dal fails on CreateCheckReference")
	})

	Context("verifyMemberExistence", func() {
		PIt("should return nil once we detect a member joining the cluster")
		PIt("should error if we exceed the 2*HeartbeatInterval wait without member joining the cluster")
		PIt("should ignore any other actions besides 'set' and 'update'")
	})

	Context("runCheckConfigWatcher", func() {
		PIt("should perform handleCheckConfigChange on an event in etcd under monitor/*")
		PIt("should log warning and break loop if not director")
		PIt("should log warning and break loop if context is cancelled")
		PIt("should log error and continue if watcher returns unexpected error")
		PIt("should ignore ignorable watcher events")
		PIt("should log error if handleCheckConfigChange fails")
	})

	Context("handleCheckConfigChange", func() {
		Context("happy path", func() {
			PIt("should create check reference on 'set' action")
			PIt("should create check reference on 'update' action")
			PIt("should create check reference on 'create' action")
			PIt("should clear check reference on 'delete' action")
			PIt("should pick the next, least 'busy' member")
		})

		PIt("should error when unable to fetch all member refs")
		PIt("should error on unrecognized action")
		PIt("should error if create/clear reference fails")
	})

	Context("PickNextMember", func() {
		PIt("should return member with the least number of assigned checks")
		PIt("should return own memberID if check stats aren't populated")
	})

	Context("ignorableWatcherEvent", func() {
		PIt("should return true if given response is nil")
		PIt("should return true if response key is ignorable")
		PIt("should return false if response key is not ignorable")
	})

	Context("setState", func() {
		PIt("should set correct bool state")
	})

	Context("amDirector", func() {
		PIt("should return director bool state")
	})
})
