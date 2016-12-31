package rye

import (
	"fmt"
	"net"
	"net/http"
)

type cidr struct {
	cidrs []string
}

/*
NewMiddlewareCIDR creates a new handler to verify incoming IPs against a set of CIDR Notation strings in a rye chain.
For reference on CIDR notation see https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing

Example usage:

	routes.Handle("/some/route", a.Dependencies.MWHandler.Handle(
		[]rye.Handler{
			rye.NewMiddlewareCIDR(CIDRs), // []string of allowed CIDRs
			yourHandler,
		})).Methods("POST")
*/
func NewMiddlewareCIDR(CIDRs []string) func(rw http.ResponseWriter, req *http.Request) *Response {
	c := &cidr{cidrs: CIDRs}
	return c.handle
}

// Verify if incoming request comes from a valid CIDR
func (c *cidr) handle(rw http.ResponseWriter, r *http.Request) *Response {
	// Validate the incoming IP
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return &Response{
			Err:        fmt.Errorf("Remote address error: %v", err.Error()),
			StatusCode: http.StatusUnauthorized,
		}
	}

	included, err := inCIDRs(host, c.cidrs)
	if err != nil {
		return &Response{
			Err:        fmt.Errorf("Error validating IP address: %v", err.Error()),
			StatusCode: http.StatusUnauthorized,
		}
	}

	if !included {
		return &Response{
			Err:        fmt.Errorf("%v is not authorized", host),
			StatusCode: http.StatusUnauthorized,
		}
	}

	return nil
}

// Verify that a given IP is a part of at least one CIDR in given CIDR list
func inCIDRs(ipAddr string, cidrList []string) (bool, error) {
	for _, v := range cidrList {
		state, err := inCIDR(ipAddr, v)
		if err != nil {
			return false, err
		}

		if state {
			return true, nil
		}
	}

	return false, nil
}

// Verify whether a given IP is in a CIDR
func inCIDR(ipAddr, cidrAddr string) (bool, error) {
	_, cidrnet, err := net.ParseCIDR(cidrAddr)
	if err != nil {
		return false, err
	}

	ip := net.ParseIP(ipAddr)

	if ip == nil {
		return false, fmt.Errorf("Unable to parse IP %v", ip)
	}

	return cidrnet.Contains(ip), nil
}
