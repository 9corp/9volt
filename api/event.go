package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/InVisionApp/rye"

	_ "github.com/9corp/9volt/event" // importing for swagger
)

// @Title Fetch Events
// @Description Fetch event data (optionally filtered by one or more event types)
// @Accept  json
// @Param   type     query    string     false        "comma separated event types"
// @Success 200 {array}  event.Event
// @Failure 500 {object} rye.JSONStatus
// @Router /event [get]
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
