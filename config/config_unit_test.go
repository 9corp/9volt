// +build unit

package config

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/9corp/9volt/dal/dalfakes"
)

var _ = Describe("ValidateDirs", func() {
	var (
		fakeDalClient *dalfakes.FakeIDal
		cfg           *Config

		testListenAddress = "0.0.0.0:8080"
		testEtcdPrefix    = "9volt"
		testEtcdMembers   = []string{"http://127.0.0.1:2379", "http://127.0.0.2:2379"}
	)

	BeforeEach(func() {
		fakeDalClient = &dalfakes.FakeIDal{}
		cfg = New(testListenAddress, testEtcdPrefix, testEtcdMembers, fakeDalClient)
	})

	Context("blah", func() {
		It("blah", func() {
			Expect(1).To(Equal(1))
		})
	})
})
