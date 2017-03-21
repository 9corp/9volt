package dal

import (
	"strings"
	"time"

	"github.com/9corp/9volt/fakes/etcdclientfakes"
	"github.com/coreos/etcd/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	"errors"
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

			Context("key has trailing slash", func() {
				BeforeEach(func() {
					testKey = "/foo/bar/path/"
				})

				It("handles properly", func() {
					Expect(err).ToNot(HaveOccurred())

					_, gotKey, _, _ := fakeKeysAPI.SetArgsForCall(0)
					Expect(gotKey).To(Equal(testDAL.Prefix + testKey[:len(testKey)-1]))
				})
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

			Context("nil options", func() {
				BeforeEach(func() {
					testOpts = nil
				})

				It("does not error", func() {
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("set returns error", func() {
			Context("ErrorCodeKeyNotFound", func() {
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

			Context("non-etcd error", func() {
				BeforeEach(func() {
					fakeKeysAPI.SetReturns(nil, errors.New("some error"))
				})

				It("returns error unmodified", func() {
					Expect(err).To(HaveOccurred())
					_, ok := err.(client.Error)
					Expect(ok).ToNot(BeTrue())
					Expect(err.Error()).To(ContainSubstring("some error"))
				})
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
					testOpts.Depth = 1

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
				})

				It("creates depth number of dirs", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeKeysAPI.SetCallCount()).To(Equal(3))
					Expect(len(setPaths)).To(Equal(3))
					Expect(setPaths).To(And(
						ContainElement(testDAL.Prefix+"/one/two"),
						ContainElement(testDAL.Prefix+"/one"),
					))
				})

				Context("depth > actual path length", func() {
					BeforeEach(func() {
						testKey = "/foo/bar/baz"
						testOpts.Depth = 8 //max is actually 2
					})

					It("creates as many parents as it can", func() {
						Expect(err).ToNot(HaveOccurred())
						Expect(fakeKeysAPI.SetCallCount()).To(Equal(5))
						Expect(len(setPaths)).To(Equal(4))
						Expect(setPaths).To(And(
							ContainElement(testDAL.Prefix+"/foo/bar/baz"),
							ContainElement(testDAL.Prefix+"/foo/bar"),
							ContainElement(testDAL.Prefix+"/foo"),
						))
					})

				})
			})

			Context("depth is < 0", func() {
				var (
					setPaths []string
				)

				BeforeEach(func() {
					testOpts.Depth = -1
					testKey = "/one/two/three"

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
				})

				It("creates all dirs", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeKeysAPI.SetCallCount()).To(Equal(5))
					Expect(len(setPaths)).To(Equal(4))
					Expect(setPaths).To(And(
						ContainElement(testDAL.Prefix+"/one/two/three"),
						ContainElement(testDAL.Prefix+"/one/two"),
						ContainElement(testDAL.Prefix+"/one"),
					))
				})
			})
		})
	})

	DescribeTable("fixDepth",
		func(d int, p string, exp int) {
			result := fixDepth(d, p)
			Expect(result).To(Equal(exp))
		},
		Entry("depth == path len",
			1, "a/b", // inputs
			1, // expected
		),
		Entry("depth < path len",
			1, "a/b/c",
			1,
		),
		Entry("depth > path len",
			3, "a/b/c",
			2,
		),
		Entry("depth < 0",
			-1, "a/b/c",
			2,
		),
		Entry("path has no /",
			3, "foo",
			0,
		),
		Entry("depth < 0 and path has no slash",
			-5, "foo",
			0,
		),
	)
})
