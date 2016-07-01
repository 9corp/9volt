package check

import "net/http"

// HTTPCheckExecutor actually runs the HTTP check
type HTTPCheckExecutor struct {
	HTTPRequest   http.Request
	HTTPResponse  *http.Response
	HTTPClient    IHttp
	httpTransport http.Transport
}

type IHttp interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

func (e *HTTPCheckExecutor) client() IHttp {
	if e.HTTPClient == nil {
		trans := &http.Transport{}
		e.HTTPClient = &http.Client{
			Timeout:   3,
			Transport: trans,
		}
	}

	return e.HTTPClient
}

// Start will actually start the check
func (e *HTTPCheckExecutor) Start() error {
	resp, err := e.client().Do(&e.HTTPRequest)
	e.HTTPResponse = resp
	return err
}

// Cancels the inflight HTTP request
func (e *HTTPCheckExecutor) Stop() error {
	e.httpTransport.CancelRequest(&e.HTTPRequest)
	return nil
}
