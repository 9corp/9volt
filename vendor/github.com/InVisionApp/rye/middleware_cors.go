package rye

import (
	"net/http"
)

const (
	// CORS Specific constants
	DEFAULT_CORS_ALLOW_ORIGIN  = "*"
	DEFAULT_CORS_ALLOW_METHODS = "POST, GET, OPTIONS, PUT, DELETE"
	DEFAULT_CORS_ALLOW_HEADERS = "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Access-Token"
)

type cors struct {
	CORSAllowOrigin  string
	CORSAllowMethods string
	CORSAllowHeaders string
}

// MiddlewareCORS is the struct to represent configuration of the CORS handler.
func MiddlewareCORS() func(rw http.ResponseWriter, req *http.Request) *Response {
	c := &cors{
		CORSAllowOrigin:  DEFAULT_CORS_ALLOW_ORIGIN,
		CORSAllowMethods: DEFAULT_CORS_ALLOW_METHODS,
		CORSAllowHeaders: DEFAULT_CORS_ALLOW_HEADERS,
	}

	return c.handle
}

/*
NewMiddlewareCORS creates a new handler to support CORS functionality. You can use this middleware by specifying `rye.MiddlewareCORS()` or `rye.NewMiddlewareCORS(origin, methods, headers)`
when defining your routes.

Default CORS Values:

	DEFAULT_CORS_ALLOW_ORIGIN**: "*"
	DEFAULT_CORS_ALLOW_METHODS**: "POST, GET, OPTIONS, PUT, DELETE"
	DEFAULT_CORS_ALLOW_HEADERS**: "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Access-Token"

If you are planning to use this in production - you should probably use this middleware *with* params.

Example use case:

	routes.Handle("/some/route", a.Dependencies.MWHandler.Handle(
		[]rye.Handler{
			rye.MiddlewareCORS(), // use defaults for allowed origin, headers, methods
			yourHandler,
		})).Methods("PUT", "OPTIONS")

OR:

	routes.Handle("/some/route", a.Dependencies.MWHandler.Handle(
		[]rye.Handler{
			rye.NewMiddlewareCORS("*", "POST, GET", "SomeHeader, AnotherHeader"),
			yourHandler,
		})).Methods("PUT", "OPTIONS")
*/
func NewMiddlewareCORS(origin, methods, headers string) func(rw http.ResponseWriter, req *http.Request) *Response {
	c := &cors{
		CORSAllowOrigin:  origin,
		CORSAllowMethods: methods,
		CORSAllowHeaders: headers,
	}

	return c.handle
}

// If `Origin` header gets passed, add required response headers for CORS support.
// Return bool if `Origin` header was detected.
func (c *cors) handle(rw http.ResponseWriter, req *http.Request) *Response {
	origin := req.Header.Get("Origin")

	// Origin header not provided, nothing for CORS to do
	if origin == "" {
		return nil
	}

	rw.Header().Set("Access-Control-Allow-Origin", c.CORSAllowOrigin)
	rw.Header().Set("Access-Control-Allow-Methods", c.CORSAllowMethods)
	rw.Header().Set("Access-Control-Allow-Headers", c.CORSAllowHeaders)

	// If this was a preflight request, stop further middleware execution
	if req.Method == "OPTIONS" {
		return &Response{
			StopExecution: true,
		}
	}

	return nil
}
