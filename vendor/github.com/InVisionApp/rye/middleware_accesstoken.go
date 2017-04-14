package rye

import (
	"errors"
	"fmt"
	"net/http"
)

type accessTokens struct {
	paramName      string
	tokens         []string
	getFunc        func(string, *http.Request) string
	missingMessage string
}

/*
NewMiddlewareAccessToken creates a new handler to verify access tokens passed as a header.

Example usage:

	routes.Handle("/some/route", a.Dependencies.MWHandler.Handle(
		[]rye.Handler{
			rye.NewMiddlewareAccessToken(tokenHeaderName, []string{token1, token2}),
			yourHandler,
		})).Methods("POST")
*/
func NewMiddlewareAccessToken(headerName string, tokens []string) func(rw http.ResponseWriter, req *http.Request) *Response {
	return newAccessTokenHandler(headerName, tokens, "header")
}

/*
NewMiddlewareAccessQueryToken creates a new handler to verify access tokens passed as a query parameter.

Example usage:

	routes.Handle("/some/route", a.Dependencies.MWHandler.Handle(
		[]rye.Handler{
			rye.NewMiddlewareAccessQueryToken(queryParamName, []string{token1, token2}),
			yourHandler,
		})).Methods("POST")
*/
func NewMiddlewareAccessQueryToken(queryParamName string, tokens []string) func(rw http.ResponseWriter, req *http.Request) *Response {
	return newAccessTokenHandler(queryParamName, tokens, "query")
}

func newAccessTokenHandler(name string, tokens []string, tokenType string) func(rw http.ResponseWriter, req *http.Request) *Response {
	a := &accessTokens{
		paramName: name,
		tokens:    tokens,
	}

	switch tokenType {

	case "query":
		a.getFunc = func(s string, r *http.Request) string {
			q, ok := r.URL.Query()[s]
			if !ok {
				return ""
			}

			return q[0]
		}
		a.missingMessage = fmt.Sprintf("No access token found; ensure you pass the '%s' parameter", name)

	default:
		// default to using the header
		a.getFunc = func(s string, r *http.Request) string {
			return r.Header.Get(s)
		}
		a.missingMessage = fmt.Sprintf("No access token found; ensure you pass '%s' in header", name)
	}

	return a.handle
}

func (a *accessTokens) handle(rw http.ResponseWriter, r *http.Request) *Response {
	token := a.getFunc(a.paramName, r)

	if token == "" {
		return &Response{
			Err:        errors.New(a.missingMessage),
			StatusCode: http.StatusUnauthorized,
		}
	}

	if ok := stringListContains(a.tokens, token); !ok {
		return &Response{
			Err:        errors.New("Unauthorized request: invalid access token"),
			StatusCode: http.StatusUnauthorized,
		}
	}

	return nil
}

func stringListContains(stringList []string, element string) bool {
	for _, v := range stringList {
		if v == element {
			return true
		}
	}

	return false
}
