package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/InVisionApp/rye"
	"github.com/coreos/etcd/client"
	"github.com/gorilla/mux"

	"github.com/9corp/9volt/alerter"
	"github.com/9corp/9volt/cfgutil"
	"github.com/9corp/9volt/dal"
)

type fullAlerterConfig map[string]*json.RawMessage

// @Title Fetch Alerter Configuration
// @Description Fetch all (or specific) alerter configuration(s) from etcd
// @Accept  json
// @Param   check     path    string     false        "Specific check name"
// @Success 200 {array}  fullAlerterConfig
// @Failure 500 {object} rye.JSONStatus
// @Router /alerter/{check} [get]
func (a *Api) AlerterHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	data, err := a.Config.DalClient.Get("alerter", &dal.GetOptions{
		Recurse: true,
	})

	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to fetch alerter configuration: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Convert every value in returned data to a json.RawMessage
	fmc := make(fullAlerterConfig, len(data))

	for k, v := range data {
		tmp := json.RawMessage(v)
		fmc[k] = &tmp
	}

	jsonData, err := json.Marshal(fmc)
	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to marshal alerter configuration: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONResponse(rw, http.StatusOK, jsonData)

	return nil
}

// Add/Update alerter config
func (a *Api) AlerterAddHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)

	var alerters = map[string]alerter.AlerterConfig{}
	if err := dec.Decode(&alerters); err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to complete config parsing: %v", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	finalAlerters := map[string][]byte{}

	for k, v := range alerters {
		alerter, err := json.Marshal(v)
		if err != nil {
			return &rye.Response{
				Err:        fmt.Errorf("Unable to complete config parsing: %v", err),
				StatusCode: http.StatusBadRequest,
			}
		}

		finalAlerters[k] = alerter
	}

	pushed, skipped, err := a.Config.DalClient.PushConfigs(cfgutil.ALERTER_TYPE, finalAlerters)
	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to complete config push: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONStatus(rw, "ok", fmt.Sprintf("Pushed %v configs; skipped %v configs", pushed, skipped), http.StatusOK)

	return nil
}

// Delete alerter config
func (a *Api) AlerterDeleteHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	alerterName := mux.Vars(r)["alerterName"]

	if alerterName == "" {
		return &rye.Response{
			Err:        errors.New("Alerter name not found. Bug?"),
			StatusCode: http.StatusInternalServerError,
		}
	}

	fullPath := fmt.Sprintf("alerter/%v", alerterName)

	if err := a.Config.DalClient.Delete(fullPath, false); err != nil {
		if client.IsKeyNotFound(err) {
			return &rye.Response{
				Err:        fmt.Errorf("Unable to find any alerter named '%v'", alerterName),
				StatusCode: http.StatusNotFound,
			}
		}

		return &rye.Response{
			Err:        fmt.Errorf("Unexpected etcd error: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONStatus(rw, "ok", fmt.Sprintf("Successfully removed alerter '%v'", alerterName), http.StatusOK)

	return nil
}

func (a *Api) AlerterGetHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	alerterName := mux.Vars(r)["alerterName"]

	if alerterName == "" {
		return &rye.Response{
			Err:        errors.New("Alerter name not found. Bug?"),
			StatusCode: http.StatusInternalServerError,
		}
	}

	fullPath := fmt.Sprintf("alerter/%v", alerterName)

	entry, err := a.Config.DalClient.Get(fullPath, nil)
	if err != nil {
		if client.IsKeyNotFound(err) {
			return &rye.Response{
				Err:        fmt.Errorf("Unable to find any alerter named '%v'", alerterName),
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
