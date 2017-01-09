package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/InVisionApp/rye"
)

// @Title Fetch Cluster Stats
// @Description Fetches cluster state data (membership, current director, heartbeats)
// @Success 200 {object} dal.ClusterStats
// @Failure 500 {object} rye.JSONStatus
// @Router /cluster [get]
func (a *Api) ClusterHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	clusterStats, err := a.Config.DalClient.GetClusterStats()
	if err != nil {
		return &rye.Response{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}

	data, err := json.Marshal(clusterStats)
	if err != nil {
		return &rye.Response{
			Err:        fmt.Errorf("Unable to marshal cluster stats: %v", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	rye.WriteJSONResponse(rw, http.StatusOK, data)
	return nil
}
