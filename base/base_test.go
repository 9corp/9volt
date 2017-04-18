package base

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("base", func() {
	Context("Identify", func() {
		It("should return identifier", func() {

			cmp := &Component{
				Identifier: "testing",
			}

			Expect(cmp.Identify()).To(Equal("testing"))
		})
	})
})
