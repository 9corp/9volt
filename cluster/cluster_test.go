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
		Context("happy path", func() {
			PIt("should create initial member structure")
			PIt("should send an initFinished notification")
			PIt("should set member state")
			PIt("should refresh its own member dir")
		})

		PIt("should error if unable to create initial member structure")
		PIt("should add event and log error if unable to generate member state")
		PIt("should add event and log error if dal fails to save member state")
		PIt("should add event and log error if dal fails to refresh member dir")
	})

	Context("generateMemberJSON", func() {
		PIt("should return a valid member state JSON blob")

		// Not sure how this can be tested exactly, as json.Marshal is tested by
		// passing it a []interface{} with some math.* values.
		//
		// https://golang.org/src/encoding/json/encode_test.go
		PIt("should return error if unable to marshal member state struct")
	})

	Context("getState", func() {
		Context("happy path", func() {
			By("having no existing director state")
			PIt("should return nil *DirectorJSON and no error")

			By("having existing director state")
			PIt("should return a pointer to DirectorJSON and no error")
		})

		PIt("should return error if dal fails to fetch state")
		PIt("should return error if returned state does not contain director key")
		PIt("should return error if unmarshalling director state json blob fails")
	})

	Context("handleState", func() {
		// TODO: remember to ensure that NOOP, CREATE or UPDATE is being used
		PIt("should become director when director json struct is nil")
		PIt("should become director if internal director state is false but etcd says otherwise")
		PIt("should step down as director if internal director state is true but etcd says otherwise")
		PIt("should become director if existing director is expired")
		PIt("should not do anything if existing director is not expired")
	})

	Context("changeState", func() {
		By("having action set to 'START'")
		PIt("should change director state to 'true'")

		By("having action set to 'STOP'")
		PIt("should change director state to 'false'")

		By("having etcdAction set to NOOP")
		PIt("should update internal director state but NOT in etcd")
	})

	Context("setDirectorState", func() {
		PIt("should update internal director state")
		PIt("should send director state change via state channel")
	})

	Context("updateState", func() {
		Context("happy path", func() {
			By("having etcdAction set to 'UPDATE'")
			PIt("should update director state")

			By("having etcdAction set to 'CREATE'")
			PIt("should create new director state")
		})

		PIt("should error if given an action other than 'CREATE' or 'UPDATE'")
		PIt("should error if unable to marshal new director state")
		PIt("should error if update via dal fails")
		PIt("should error if create via dal fails")
	})

	Context("isExpired", func() {
		PIt("should return true if given datetime is older than NOW+HeartbeatTimeout")
		PIt("should return false if given datetime is NOT older than Now()+HeartbeatTimeout")
	})

	Context("amDirector", func() {
		PIt("should return true if current director state is true")
		PIt("should return false if current director state is false")
	})
})
