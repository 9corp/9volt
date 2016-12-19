package monitor

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

type HTTPMonitor struct {
	Base
}

func NewHTTPMonitor(rmc *RootMonitorConfig) IMonitor {
	h := &HTTPMonitor{
		Base: Base{
			RMC:        rmc,
			Identifier: "http",
		},
	}

	h.MonitorFunc = h.httpCheck

	return h
}

func (h *HTTPMonitor) Validate() error {
	return nil
}

// Perform a statusCode check; optionally, if 'Expect' is not blank, verify that
// the received response body contains the 'Expect' string.
func (h *HTTPMonitor) httpCheck() error {
	fullURL := h.constructURL()

	log.Debugf("%v-%v: Performing http check for '%v'", h.Identify(), h.RMC.GID, fullURL)

	resp, err := h.performRequest(h.RMC.Config.HTTPMethod, fullURL, h.RMC.Config.HTTPRequestBody)
	if err != nil {
		return err
	}

	// Check if StatusCode matches
	if resp.StatusCode != h.RMC.Config.HTTPStatusCode {
		return fmt.Errorf("Received status code '%v' does not match expected status code '%v'",
			resp.StatusCode, h.RMC.Config.HTTPStatusCode)
	}

	// If Expect is set, verify if returned response contains expected data
	if h.RMC.Config.Expect != "" {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Unable to read response body to perform content expectancy check: %v", err.Error())
		}
		defer resp.Body.Close()

		if !strings.Contains(string(data), h.RMC.Config.Expect) {
			return fmt.Errorf("Received response body '%v' does not contain expected content '%v'",
				string(data), h.RMC.Config.Expect)
		}
	}

	return nil
}

func (h *HTTPMonitor) constructURL() string {
	fullURL := "http://"

	// Use http:// or https://
	if h.RMC.Config.HTTPSSL {
		fullURL = "https://"
	}

	fullURL = fullURL + h.RMC.Config.Host

	// If port is set, tack on a ':PORT'
	if h.RMC.Config.Port != 0 {
		fullURL = fmt.Sprintf("%v:%v", fullURL, h.RMC.Config.Port)
	}

	// If URL does not start with '/', tack it on
	if !strings.HasPrefix(h.RMC.Config.HTTPURL, "/") {
		fullURL = fullURL + "/"
	}

	// Return the constructed URL
	fullURL = fullURL + h.RMC.Config.HTTPURL

	return fullURL
}

// Create and perform a new HTTP request with a timeout; return http Response
func (h *HTTPMonitor) performRequest(method, urlStr, requestBody string) (*http.Response, error) {
	client := &http.Client{
		Timeout: time.Duration(h.RMC.Config.Timeout),
	}

	// TODO: Not sure if it's okay to just `body := strings.NewReader(requestBody)`,
	//       even if the requestBody is empty.
	var body *strings.Reader
	body = nil

	if requestBody == "" {
		body = strings.NewReader(requestBody)
	}

	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("Unable to create new HTTP request for HTTPMonitor check: %v", err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ran into error while performing '%v' request: %v", method, err.Error())
	}

	return resp, nil
}
