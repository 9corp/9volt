package util

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("util", func() {
	Context("CustomDuration", func() {
		PIt("should be able to unmarshal JSON with 'CustomDuration'")
		PIt("should error unmarshalling bad JSON input")
		PIt("should error on bad duration input")
	})

	Context("MD5Hash", func() {
		PIt("should return hash as long as length")
		PIt("should return full hash if hash smaller than length")
	})

	Context("RandomString", func() {
		PIt("should return a specific length random string")
		PIt("should seed if seed is true")
	})

	Context("GetMemberID", func() {
		PIt("should return memberid")
	})

	Context("StringSliceContains", func() {
		PIt("should be true if slice contains given string")
		PIt("should be false if slice does not contain given string")
	})

	Context("StringSliceInStringSlice", func() {
		PIt("shoudl be true if a string in slice A exists in slice B")
		PIt("should be false if a string in slice A does not exist in slice B")
	})
})
