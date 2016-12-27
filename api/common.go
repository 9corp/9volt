package api

import (
	"fmt"
	"net/http"

	"github.com/InVisionApp/rye"
)

func (a *Api) HomeHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	fmt.Fprint(rw, "Refer to README.md for 9volt API usage")
	return nil
}

func (a *Api) StatusHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	rye.WriteJSONStatus(rw, "OK", fmt.Sprintf("MemberID %v", a.MemberID), http.StatusOK)
	return nil
}

func (a *Api) VersionHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	rye.WriteJSONStatus(rw, "version", fmt.Sprintf("9volt %v", a.Version), http.StatusOK)
	return nil
}
