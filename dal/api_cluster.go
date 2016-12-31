package dal

import (
	"encoding/json"
	"fmt"
)

type ClusterStats struct {
	Members  map[string]*json.RawMessage
	Director *json.RawMessage
}

func (d *Dal) GetClusterStats() (*ClusterStats, error) {
	// get all members
	members, err := d.GetClusterMembers()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch cluster member info: %v", err)
	}

	clusterStats := &ClusterStats{
		Members: make(map[string]*json.RawMessage, 0),
	}

	// fetch status for each member
	for _, v := range members {
		fullPath := "cluster/members/" + v + "/status"

		memberStatus, err := d.Get(fullPath, nil)
		if err != nil {
			return nil, fmt.Errorf("Unable to fetch member info for '%v': %v", v, err)
		}

		memberBlob := json.RawMessage(memberStatus[fullPath])
		clusterStats.Members[v] = &memberBlob
	}

	// fetch director status (/cluster/director)
	directorStatus, err := d.Get("cluster/director", nil)
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch director status: %v", err)
	}

	clusterBlob := json.RawMessage(directorStatus["cluster/director"])
	clusterStats.Director = &clusterBlob

	return clusterStats, nil
}
