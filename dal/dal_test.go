package dal

import (
	"github.com/9corp/9volt/fakes/etcdclientfakes"
	"github.com/coreos/etcd/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	"strings"
	"time"
)

var _ = Describe("dal", func() {
	var (
		testDAL     *Dal
		fakeKeysAPI *etcdclientfakes.FakeKeysAPI
	)

	BeforeEach(func() {
		fakeKeysAPI = &etcdclientfakes.FakeKeysAPI{}
		testDAL = &Dal{
			KeysAPI: fakeKeysAPI,
			Prefix:  "testpre",
		}
	})

	Describe("Set", func() {
		var (
			testKey, testVal string
			testOpts         *SetOptions
			err              error
		)

		BeforeEach(func() {
			testKey, testVal = "foo/bar/path", "baz"
			testOpts = &SetOptions{
				Dir:       false,
				TTLSec:    5,
				PrevExist: "",
			}
		})

		JustBeforeEach(func() {
			err = testDAL.Set(testKey, testVal, testOpts)
		})

		Context("happy path", func() {
			BeforeEach(func() {
				fakeKeysAPI.SetReturns(&client.Response{}, nil)
			})

			It("does not error", func() {
				Expect(err).ToNot(HaveOccurred())

				_, gotKey, gotVal, gotOpts := fakeKeysAPI.SetArgsForCall(0)
				Expect(gotKey).To(Equal(testDAL.Prefix + "/" + testKey))
				Expect(gotVal).To(Equal(testVal))
				Expect(gotOpts.Dir).To(Equal(testOpts.Dir))
				Expect(gotOpts.TTL).To(Equal(time.Duration(testOpts.TTLSec) * time.Second))
				Expect(gotOpts.PrevExist).To(Equal(client.PrevExistType(testOpts.PrevExist)))
			})

			Context("key has leading slash", func() {
				BeforeEach(func() {
					testKey = "/foo/bar/path"
				})

				It("handles properly", func() {
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
				Expect(err).To(HaveOccurred())
				etcdErr, ok := err.(client.Error)
				Expect(ok).To(BeTrue())
				Expect(etcdErr.Code).To(Equal(client.ErrorCodeKeyNotFound))
			})
		})

		Context("create parents is set", func() {
			BeforeEach(func() {
				testOpts.CreateParents = true
			})

			Context("depth is 0", func() {
				BeforeEach(func() {
					fakeKeysAPI.SetReturns(&client.Response{}, nil)
					testOpts.Depth = 0
				})

				It("sets single item only", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeKeysAPI.SetCallCount()).To(Equal(1))
				})
			})

			Context("depth is > 0", func() {
				var (
					setPaths []string
				)

				BeforeEach(func() {
					testKey = "/one/two"
					setPaths = []string{testDAL.Prefix}

					fakeKeysAPI.SetStub = func(ctx context.Context, key, value string, opts *client.SetOptions) (*client.Response, error) {
						parent := key[:strings.LastIndex(key, "/")]
						for _, p := range setPaths {
							if p == parent {
								setPaths = append(setPaths, key)
								return &client.Response{}, nil
							}
						}

						return nil, client.Error{Code: client.ErrorCodeKeyNotFound}
					}

					testOpts.Depth = 1
				})

				It("creates depth number of dirs", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeKeysAPI.SetCallCount()).To(Equal(3))
					Expect(setPaths).To(And(
						ContainElement(testDAL.Prefix+testKey),
						ContainElement(testDAL.Prefix+testKey[:strings.LastIndex(testKey, "/")]),
					))
				})

				Context("depth > actual path length", func() {
					//TODO this will probably error currently, so definitely need to do it
				})
			})

			Context("depth is < 0", func() {
				BeforeEach(func() {
					testOpts.Depth = -1
				})

				It("creates all dirs", func() {
					//TODO
				})
			})
		})
	})
})
