package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/InVisionApp/rye"
	// log "github.com/Sirupsen/logrus"
)

func (a *Api) StateHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	stateData, err := a.Config.DalClient.FetchState()
	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to fetch state data: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONResponse(rw, http.StatusOK, stateData)
	return nil
}

func (a *Api) StateWithTagsHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	vals := r.URL.Query()
	if _, ok := vals["tags"]; !ok {
		rye.WriteJSONStatus(rw, "error", "No tags found", http.StatusBadRequest)
		return nil
	}

	tags := strings.Split(vals["tags"][0], ",")

	stateData, err := a.Config.DalClient.FetchStateWithTags(tags)
	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to fetch state data: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONResponse(rw, http.StatusOK, stateData)
	return nil
}
