package rye

import (
	"net/http"
)

type staticFilesystem struct {
	path        string
	stripPrefix string
}

/*
NewStaticFilesystem creates a new handler to serve a filesystem from a path
on the local filesystem. The path should be an absolute path -> i.e., it's
up to the program using Rye to correctly determine what path it should be
serving from. An example is available in the `static_example.go` file which
shows setting up a path relative to the go executable.

The primary benefit of this is to serve an entire set of files. You can
pre-pend typical Rye middlewares to the chain. The static filesystem
middleware should always be last in a chain, however. The `stripPrefix` allows
you to ignore the prefix on requests so that the proper files will be matched.

Example use case:

	routes.PathPrefix("/dist/").Handler(middlewareHandler.Handle([]rye.Handler{
		rye.MiddlewareRouteLogger(),
		rye.NewStaticFilesystem(pwd+"/dist/", "/dist/"),
	}))

*/
func NewStaticFilesystem(path string, stripPrefix string) func(rw http.ResponseWriter, req *http.Request) *Response {
	s := &staticFilesystem{
		path:        path,
		stripPrefix: stripPrefix,
	}
	return s.handle
}

func (s *staticFilesystem) handle(rw http.ResponseWriter, req *http.Request) *Response {
	x := http.StripPrefix(s.stripPrefix, http.FileServer(http.Dir(s.path)))
	x.ServeHTTP(rw, req)
	return nil
}
