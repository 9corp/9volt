package check

import (
	"testing"

	fake "github.com/9corp/9volt/check/checkfakes"
)

func createHTTPExecutor() (HTTPCheckExecutor, *fake.FakeIHttp) {
	dummy := &fake.FakeIHttp{}
	exec := HTTPCheckExecutor{
		HTTPClient: dummy,
	}

	return exec, dummy
}

func TestHTTPStart(t *testing.T) {
	h, c := createHTTPExecutor()

	if h.Start() != nil {
		t.Fail()
	}

	if c.DoCallCount() == 0 {
		t.Fail()
	}
}

func TestHTTPStop(t *testing.T) {
	h, _ := createHTTPExecutor()

	if h.Stop() != nil {
		t.Fail()
	}
}
