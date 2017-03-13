package dal

import (
	"github.com/9corp/9volt/fakes/etcdclientfakes"
	"github.com/coreos/etcd/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("dal", func() {
	var (
		testDAL     *Dal
		fakeKeysAPI *etcdclientfakes.FakeKeysAPI

		testKey, testVal, testPrevExists string
		testTTL                          int
		testDir                          bool
	)

	BeforeEach(func() {
		fakeKeysAPI = &etcdclientfakes.FakeKeysAPI{}
		testDAL = &Dal{
			KeysAPI: fakeKeysAPI,
			Prefix:  "testpre",
		}

		testKey, testVal, testDir, testTTL, testPrevExists =
			"foo/bar/path", "baz", false, 5, ""

	})

	Describe("Set", func() {

		Context("happy path", func() {
			BeforeEach(func() {
				fakeKeysAPI.SetReturns(&client.Response{}, nil)
			})

			It("does not error", func() {
				err := testDAL.Set(testKey, testVal, &SetOptions{Dir: testDir, TTLSec: testTTL, PrevExist: testPrevExists})
				Expect(err).ToNot(HaveOccurred())

				_, gotKey, gotVal, gotOpts := fakeKeysAPI.SetArgsForCall(0)
				Expect(gotKey).To(Equal(testDAL.Prefix + "/" + testKey))
				Expect(gotVal).To(Equal(testVal))
				Expect(gotOpts.Dir).To(Equal(testDir))
				Expect(gotOpts.TTL).To(Equal(time.Duration(testTTL) * time.Second))
				Expect(gotOpts.PrevExist).To(Equal(client.PrevExistType(testPrevExists)))
			})

			Context("key has leading slash", func() {
				BeforeEach(func() {
					testKey = "/foo/bar/path"
				})

				It("handles properly", func() {
					err := testDAL.Set(testKey, testVal, &SetOptions{Dir: testDir, TTLSec: testTTL, PrevExist: testPrevExists})
					Expect(err).ToNot(HaveOccurred())

					_, gotKey, _, _ := fakeKeysAPI.SetArgsForCall(0)
					Expect(gotKey).To(Equal(testDAL.Prefix + testKey))
				})
			})
		})

		Context("set returns error", func() {
			BeforeEach(func() {
				fakeKeysAPI.SetReturns(nil, client.Error{Code: client.ErrorCodeKeyNotFound})
			})

			It("returns error unmodified", func() {
				err := testDAL.Set(testKey, testVal, &SetOptions{Dir: testDir, TTLSec: testTTL, PrevExist: testPrevExists})
				Expect(err).To(HaveOccurred())
				etcdErr, ok := err.(client.Error)
				Expect(ok).To(BeTrue())
				Expect(etcdErr.Code).To(Equal(client.ErrorCodeKeyNotFound))
			})
		})
	})
})
