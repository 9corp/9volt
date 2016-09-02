package check

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func handler(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte("Testing"))
}

func failureHandler(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusInternalServerError)
	resp.Write([]byte("Error"))
}

var ts *httptest.Server

func createHTTPExecutor() *HTTPCheckExecutor {
	exec := &HTTPCheckExecutor{}
	ts = httptest.NewServer(http.HandlerFunc(handler))
	exec.URL = ts.URL

	return exec
}

func TestHTTPStart(t *testing.T) {
	h := createHTTPExecutor()

	h.Start()
}

func TestFailed(t *testing.T) {
	ets := httptest.NewServer(http.HandlerFunc(failureHandler))
	h := createHTTPExecutor()

	h.URL = ets.URL

	h.Start()

	if h.Failed() != true {
		t.Errorf("HTTP executor Failed() should have returned true instead returned: %t\n", h.Failed())
	}
}
