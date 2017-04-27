package monitor

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/9corp/9volt/util"
)

const (
	DEFAULT_HTTP_TIMEOUT = time.Duration(3) * time.Second
)

type HTTPMonitor struct {
	Base

	Timeout time.Duration
}

func NewHTTPMonitor(rmc *RootMonitorConfig) *HTTPMonitor {
	h := &HTTPMonitor{
		Base: Base{
			RMC:        rmc,
			Identifier: "http",
		},
	}

	if rmc.Config.Timeout == util.CustomDuration(0) {
		h.Timeout = DEFAULT_HTTP_TIMEOUT
	} else {
		h.Timeout = time.Duration(rmc.Config.Timeout)
	}

	h.MonitorFunc = h.httpCheck

	return h
}

func (h *HTTPMonitor) Validate() error {
	h.RMC.Log.WithField("configName", h.RMC.ConfigName).Debug("Performing monitor config validation")

	if h.Timeout >= time.Duration(h.RMC.Config.Interval) {
		return fmt.Errorf("'timeout' (%v) cannot equal or exceed 'interval' (%v)", h.Timeout.String(), h.RMC.Config.Interval.String())
	}

	return nil
}

// Perform a statusCode check; optionally, if 'Expect' is not blank, verify that
// the received response body contains the 'Expect' string.
func (h *HTTPMonitor) httpCheck() error {
	fullURL := h.constructURL()

	h.RMC.Log.WithField("fullURL", fullURL).Debug("Performing http check")

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
	scheme := "http"
	if h.RMC.Config.HTTPSSL {
		scheme = "https"
	}

	checkUrl := url.URL{
		Scheme: scheme,
		Host:   h.RMC.Config.Host,
	}

	// If port is set, tack on a ':PORT'
	if h.RMC.Config.Port != 0 {
		checkUrl.Host = fmt.Sprintf("%s:%d", checkUrl.Host, h.RMC.Config.Port)
	}

	checkUrl.Path = "/" + strings.TrimLeft(h.RMC.Config.HTTPURL, "/")

	return checkUrl.String()
}

// Create and perform a new HTTP request with a timeout; return http Response
func (h *HTTPMonitor) performRequest(method, urlStr, requestBody string) (*http.Response, error) {

	client := &http.Client{Timeout: h.Timeout}

	body := strings.NewReader(requestBody)

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
