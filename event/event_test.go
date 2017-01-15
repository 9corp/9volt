package event

import (
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("event", func() {
	Context("NewQueue", func() {
		PIt("should create and return new instance of Queue")
	})

	Describe("*Queue", func() {
		Context("Start", func() {
			PIt("should start required number of workers")
		})

		Context("runWorker", func() {
			Context("happy path", func() {
				PIt("should get event message")
				PIt("should save the event via dal")
			})

			Context("error path", func() {
				PIt("should log an error if unable to marshal event blob")
				PIt("shoudl log an error if dal returns an error")
			})
		})

		Context("NewQueue", func() {
			PIt("should return a new Client instance with current queue")
		})
	})

	Describe("*Client", func() {
		Context("Add", func() {
			PIt("should return nil if channel send succeeds")
			PIt("should return an error if channel is full")
		})

		Context("AddWithErrorLog", func() {
			PIt("should log error and return nil")
			PIt("should return error if Add() fails")
		})
	})
})
