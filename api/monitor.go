package api

import (
	"fmt"
	"net/http"

	"github.com/InVisionApp/rye"
)

func (a *Api) MonitorHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	fmt.Fprint(rw, "monitor handler")
	return nil
}

func (a *Api) MonitorDisableHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	fmt.Fprint(rw, "monitor disable handler")
	return nil
}

func (a *Api) MonitorCheckHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	fmt.Fprint(rw, "monitor check handler")
	return nil
}
