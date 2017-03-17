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
	ok, msg := a.Config.Health.Read()
	memberID := fmt.Sprintf("MemberID %v", a.MemberID)

	if ok {
		rye.WriteJSONStatus(rw, msg, memberID, http.StatusOK)
	} else {
		rye.WriteJSONStatus(rw, msg, memberID, http.StatusInternalServerError)
	}

	return nil
}

func (a *Api) VersionHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	rye.WriteJSONStatus(rw, "version", fmt.Sprintf("9volt %v - %v", a.Config.SemVer, a.Config.Version), http.StatusOK)
	return nil
}
