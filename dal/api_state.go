package dal

import (
	"encoding/json"
	"fmt"
	"reflect"

	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/util"
)

// Wrapper for fetching state information
func (d *Dal) FetchState() ([]byte, error) {
	stateData, err := d.Get("state", &GetOptions{
		Recurse: true,
	})

	if err != nil {
		return nil, err
	}

	combinedData := make([]json.RawMessage, 0)

	for _, v := range stateData {
		combinedData = append(combinedData, []byte(v))
	}

	full, err := json.Marshal(combinedData)
	if err != nil {
		return nil, fmt.Errorf("Unable to convert JSON state data: %v", err)
	}

	return full, nil
}

// Wrapper for fetching state information that is tagged with `tags`
//
// Note: To avoid having to import the MonitorConfig struct, we will inspect it
//       manually via assertions and reflect.
func (d *Dal) FetchStateWithTags(tags []string) ([]byte, error) {
	stateData, err := d.FetchState()
	if err != nil {
		return nil, err
	}

	filteredDataset := make([]json.RawMessage, 0)
	var dataset []map[string]interface{}

	if err := json.Unmarshal(stateData, &dataset); err != nil {
		return nil, fmt.Errorf("Unable to perform initial state data conversion: %v", err)
	}

	for _, entry := range dataset {
		found, err := d.stateContainsConfigWithTags(entry, tags)
		if err != nil {
			log.Errorf("Unable to determine if state entry '%v' contains our wanted tags: %v", entry, err)
			continue
		} else if !found {
			continue
		}

		// This entry contains the tags we are looking for; remarshal and populate
		// our filtered data set
		remarshalled, err := json.Marshal(entry)
		if err != nil {
			return nil, fmt.Errorf("Unable to re-marshal filtered dataset: %v", err)
		}

		filteredDataset = append(filteredDataset, remarshalled)
	}

	// final remarshal
	final, err := json.Marshal(filteredDataset)
	if err != nil {
		return nil, fmt.Errorf("Unable to perform final filtered dataset marshal: %v", err)
	}

	return final, nil
}

// Helper for determining if a state entry has a "config" with a "tags" slice
// (and if the found tags slice contains any one of the tags specified in `tags`)
func (d *Dal) stateContainsConfigWithTags(entry map[string]interface{}, tags []string) (bool, error) {
	if _, ok := entry["config"]; !ok {
		return false, nil
	}

	configs, found := entry["config"].(map[string]interface{})
	if !found {
		return false, nil
	}

	if _, ok := configs["tags"]; !ok {
		return false, nil
	}

	s := reflect.ValueOf(configs["tags"])
	if s.Kind() != reflect.Slice {
		return false, nil
	}

	// populate tags we've discovered
	foundTags := make([]string, s.Len())

	for i := 0; i < s.Len(); i++ {
		foundTags[i] = s.Index(i).Elem().String()
	}

	if util.StringSliceInStringSlice(tags, foundTags) {
		return true, nil
	}

	return false, nil
}
