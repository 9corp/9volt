package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/InVisionApp/rye"
	"github.com/coreos/etcd/client"
	"github.com/gorilla/mux"

	"github.com/9corp/9volt/dal"
)

type fullMonitorConfig map[string]*json.RawMessage

func (a *Api) MonitorHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	data, err := a.Config.DalClient.Get("monitor", &dal.GetOptions{
		Recurse: true,
	})

	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to fetch monitor configuration: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Convert every value in returned data to a json.RawMessage
	fmc := make(fullMonitorConfig, len(data))

	for k, v := range data {
		tmp := json.RawMessage(v)
		fmc[k] = &tmp
	}

	jsonData, err := json.Marshal(fmc)
	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to marshal monitor configuration: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONResponse(rw, http.StatusOK, jsonData)

	return nil
}

func (a *Api) MonitorDisableHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	fmt.Fprint(rw, "monitor disable handler")
	return nil
}

func (a *Api) MonitorCheckHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	checkName := mux.Vars(r)["check"]

	if checkName == "" {
		return &rye.Response{
			Err:        errors.New("Check name not found. Bug?"),
			StatusCode: http.StatusInternalServerError,
		}
	}

	fullPath := fmt.Sprintf("monitor/%v", checkName)

	entry, err := a.Config.DalClient.Get(fullPath, nil)
	if err != nil {
		if client.IsKeyNotFound(err) {
			return &rye.Response{
				Err:        fmt.Errorf("Unable to find any check named '%v'", checkName),
				StatusCode: http.StatusNotFound,
			}
		}

		return &rye.Response{
			Err:        fmt.Errorf("Unexpected etcd error: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	raw := json.RawMessage(entry[fullPath])

	jsonData, err := json.Marshal(&raw)
	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to marshal entry to JSON: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONResponse(rw, http.StatusOK, jsonData)

	return nil
}
