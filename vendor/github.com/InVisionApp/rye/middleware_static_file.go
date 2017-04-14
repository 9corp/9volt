package rye

import (
	"net/http"
)

type staticFile struct {
	path string
}

/*
NewStaticFile creates a new handler to serve a file from a path on the local filesystem.
The path should be an absolute path -> i.e., it's up to the program using Rye to
correctly determine what path it should be serving from. An example is available
in the `static_example.go` file which shows setting up a path relative to
the go executable.

The purpose of this handler is to serve a specific file for any requests through the
route handler. For instance, in the example below, any requests made to `/ui` will
always be routed to /dist/index.html. This is important for single page applications
which happen to use client-side routers. Therefore, you might have a webpack application
with it's entrypoint `/dist/index.html`. That file may point at your `bundle.js`.
Every request into the app will need to always be routed to `/dist/index.html`

Example use case:

	routes.PathPrefix("/ui/").Handler(middlewareHandler.Handle([]rye.Handler{
		rye.MiddlewareRouteLogger(),
		rye.NewStaticFile(pwd + "/dist/index.html"),
	}))

*/
func NewStaticFile(path string) func(rw http.ResponseWriter, req *http.Request) *Response {
	s := &staticFile{
		path: path,
	}
	return s.handle
}

func (s *staticFile) handle(rw http.ResponseWriter, req *http.Request) *Response {
	http.ServeFile(rw, req, s.path)
	return nil
}
