package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/InVisionApp/rye"
)

func (a *Api) EventHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	eventData, err := a.Config.DalClient.FetchEvents([]string{})
	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to fetch event data: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONResponse(rw, http.StatusOK, eventData)
	return nil
}

func (a *Api) EventWithTypeHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	vals := r.URL.Query()
	if _, ok := vals["type"]; !ok {
		rye.WriteJSONStatus(rw, "error", "No type found", http.StatusBadRequest)
		return nil
	}

	types := strings.Split(vals["type"][0], ",")

	eventData, err := a.Config.DalClient.FetchEvents(types)
	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to fetch event data: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONResponse(rw, http.StatusOK, eventData)
	return nil
}
