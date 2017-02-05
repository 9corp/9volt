package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	// This is required by the statik package
	_ "github.com/9corp/9volt/statik"
	"github.com/InVisionApp/rye"
	"github.com/rakyll/statik/fs"
)

// This handler serves the Dashboard: index.html from the file system
func (a *Api) uiHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	_, b, _, _ := runtime.Caller(0)
	entrypoint := fmt.Sprintf("%v/../ui/dist/index.html", filepath.Dir(b))

	http.ServeFile(rw, r, entrypoint)

	return nil
}

// This handler serves all Dashboard public artifacts from the file system
func (a *Api) uiDistHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	_, b, _, _ := runtime.Caller(0)
	entrypoint := fmt.Sprintf("%v/../ui/dist/", filepath.Dir(b))

	x := http.StripPrefix("/dist/", http.FileServer(http.Dir(entrypoint)))
	x.ServeHTTP(rw, r)
	return nil
}

// This handler serves all Dashboard public artifacts from a static file system in memory
func (a *Api) uiDistStatikHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	statikFS, err := fs.New()
	if err != nil {
		return &rye.Response{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	x := http.StripPrefix("/dist/", http.FileServer(statikFS))
	x.ServeHTTP(rw, r)
	return nil
}

// This handler serves the index.html from a static file system in memory
func (a *Api) uiStatikHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	statikFS, err := fs.New()
	if err != nil {
		return &rye.Response{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	file, err := statikFS.Open("/index.html")
	if err != nil {
		return &rye.Response{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return &rye.Response{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	http.ServeContent(rw, r, "index.html", time.Now(), bytes.NewReader(b))

	return nil
}
