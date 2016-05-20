// +build unit

package config

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Context("when test lib is setup correctly", func() {
		It("should not error", func() {
			Expect(1).To(Equal(1))
		})
	})
})
