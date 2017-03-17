package config

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/9corp/9volt/dal/dalfakes"
)

var _ = Describe("config", func() {
	var (
		fakeDalClient *dalfakes.FakeIDal
		cfg           *Config

		testMemberID      = "testMemberID"
		testListenAddress = "0.0.0.0:8080"
		testEtcdPrefix    = "9volt"
		testEtcdUserPass  = ""
		testEtcdMembers   = []string{"http://127.0.0.1:2379", "http://127.0.0.2:2379"}
		testTags          = []string{"tag1", "tag2"}
		testVersion       = "12345"
		testSemver        = "0.0.1"
	)

	BeforeEach(func() {
		fakeDalClient = &dalfakes.FakeIDal{}
		cfg = New(testMemberID, testListenAddress, testEtcdPrefix, testEtcdUserPass, testEtcdMembers,
			testTags, fakeDalClient, nil, testVersion, testSemver)

		Expect(cfg).ToNot(BeNil())
	})

	Context("ValidateDirs", func() {
		PIt("should return empty string slice on no errors")
		PIt("should error if dal runs into an error during key existance check")
		PIt("should error if required key does not exist")
		PIt("should error if required key is not a dir")
		PIt("should return a slice of errors (if errors are hit)")
	})

	Context("Load", func() {
		PIt("should load server config")
		PIt("should error if dal runs into error on initial config fetch")
		PIt("should error if config does not exist")
		PIt("should error if config is a dir (and not a key)")
		PIt("should error if dal get errors")
		PIt("should error if returned value map does not contain config")
		PIt("should error if load() returns an error")
	})

	Context("load", func() {
		PIt("should load server config")
		PIt("should error if config can't be unmarshalled onto serverConfig struct")
		PIt("should error if validate() fails")
	})

	Context("validate", func() {
		PIt("should return nil on happy path")
		PIt("should error on invalid HeartbeatInterval")
		PIt("should error on invalid HeartbeatTimeout")
		PIt("should error on invalid StateDumpInterval")
	})
})
