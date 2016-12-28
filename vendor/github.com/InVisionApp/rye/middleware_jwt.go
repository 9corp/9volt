package rye

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/dgrijalva/jwt-go"
)

const (
	CONTEXT_JWT = "rye-middlewarejwt-jwt"
)

type jwtVerify struct {
	secret string
	token  string
}

/*
This middleware provides JWT verification functionality

You can use this middleware by specifying `rye.NewMiddlewareJWT(shared_secret)`
when defining your routes.

This middleware has no default version, it must be configured with a shared secret.

Example use case:

	routes.Handle("/some/route", a.Dependencies.MWHandler.Handle(
		[]rye.Handler{
			rye.NewMiddlewareJWT("this is a big secret"),
			yourHandler,
		})).Methods("PUT", "OPTIONS")

Additionally, this middleware puts the JWT token into the context for use by other
middlewares in your chain.

Access to that is simple (using the CONTEXT_JWT constant as a key)

	func getJWTfromContext(rw http.ResponseWriter, r *http.Request) *rye.Response {

		// Retrieving the value is easy!
		// Just reference the rye.CONTEXT_JWT const as a key
		myVal := r.Context().Value(rye.CONTEXT_JWT)

		// Log it to the server log?
		log.Infof("Context Value: %v", myVal)

		return nil
	}

*/
func NewMiddlewareJWT(secret string) func(rw http.ResponseWriter, req *http.Request) *Response {
	j := &jwtVerify{secret: secret}
	return j.handle
}

func (j *jwtVerify) handle(rw http.ResponseWriter, req *http.Request) *Response {

	tokenHeader := req.Header.Get("Authorization")

	if tokenHeader == "" {
		return &Response{
			Err:        fmt.Errorf("JWT token must be passed with Authorization header"),
			StatusCode: 400,
		}
	}

	// Remove 'Bearer' prefix
	p, _ := regexp.Compile(`(?i)bearer\s+`)
	j.token = p.ReplaceAllString(tokenHeader, "")

	_, err := jwt.Parse(j.token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method")
		}
		return []byte(j.secret), nil
	})

	if err != nil {
		return &Response{
			Err:        err,
			StatusCode: 401,
		}
	}

	ctx := context.WithValue(req.Context(), CONTEXT_JWT, j.token)

	return &Response{Context: ctx}
}
