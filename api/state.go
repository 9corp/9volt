package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/InVisionApp/rye"

	"github.com/9corp/9volt/dal"
)

func (a *Api) StateHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	stateData, err := a.Config.DalClient.Get("state", &dal.GetOptions{
		Recurse: true,
	})

	if err != nil {
		rye.WriteJSONStatus(rw, "error", fmt.Sprintf("Unable to fetch state: %v", err), http.StatusInternalServerError)
		return nil
	}

	fullResponse := make([]json.RawMessage, 0)

	for _, v := range stateData {
		fullResponse = append(fullResponse, []byte(v))
	}

	data, err := json.Marshal(fullResponse)
	if err != nil {
		rye.WriteJSONStatus(rw, "error", "crap", http.StatusInternalServerError)
		return nil
	}

	rye.WriteJSONResponse(rw, http.StatusOK, data)
	return nil
}

func (a *Api) StateWithTagsHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	fmt.Fprintf(rw, "state with tags handler")
	return nil
}
