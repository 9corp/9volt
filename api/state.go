package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/InVisionApp/rye"

	_ "github.com/9corp/9volt/state"
)

// @Title Fetch Check State Data
// @Description Fetch check state data including latest check status, ownership, last check timestamp;
//              optionally filter the state data by checks that contain one or more tags.
// @Accept  json
// @Param   tags     query    string     false        "One or more tags (comma separated)"
// @Success 200 {array}  state.Message
// @Failure 500 {object} rye.JSONStatus
// @Router /state [get]
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
