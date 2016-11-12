package check

import (
	"net/http"
	"net/url"
)

// HTTPCheckExecutor actually runs the HTTP check
type HTTPCheckExecutor struct {
	URL           string
	request       http.Request
	response      *http.Response
	client        http.Client
	httpTransport http.Transport
	failed        bool
	lastError     error
}

// Start will actually start the check
func (e *HTTPCheckExecutor) Start() {
	url, err := url.Parse(e.URL)
	if err != nil {
		e.failed = true
		e.lastError = err
		return
	}
	e.request.URL = url
	resp, err := e.client.Do(&e.request)
	e.response = resp
	if err != nil {
		e.failed = true
		e.lastError = err
	}

	if !(e.response.StatusCode >= 200 && e.response.StatusCode <= 299) {
		e.failed = true
	}
}

func (e *HTTPCheckExecutor) Failed() bool {
	return e.failed
}

func (e *HTTPCheckExecutor) LastError() error {
	return e.lastError
}
