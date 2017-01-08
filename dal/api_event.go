package dal

import (
	"encoding/json"
	"fmt"
	"path"

	log "github.com/Sirupsen/logrus"
)

// Fetch event data that matches at least one type specified in 'types'; if 'types'
// is empty, do not perform any filtering.
func (d *Dal) FetchEvents(types []string) ([]byte, error) {
	eventData, err := d.Get("event", &GetOptions{
		Recurse: true,
	})

	if err != nil {
		return nil, err
	}

	combinedData := make(map[string]*json.RawMessage, len(eventData))

	for k, v := range eventData {
		if !d.eventContainsTypes(v, types) {
			continue
		}

		event := json.RawMessage(v)
		combinedData[path.Base(k)] = &event
	}

	eventBlob, err := json.Marshal(combinedData)
	if err != nil {
		return nil, fmt.Errorf("Unable to re-marshal event data: %v", err)
	}

	return eventBlob, nil
}

// Check if given event data contains at least one of the types specified in 'types'
func (d *Dal) eventContainsTypes(eventData string, types []string) bool {
	if len(types) == 0 {
		return true
	}

	var tmpJSON map[string]string
	if err := json.Unmarshal([]byte(eventData), &tmpJSON); err != nil {
		log.Debugf("Unable to complete JSON unmarshal for event type check: %v", err)
		return false
	}

	if _, ok := tmpJSON["type"]; !ok {
		log.Debug("Unmarshaled event JSON data does not appear to contain a 'type' element")
		return false
	}

	for _, t := range types {
		if t == tmpJSON["type"] {
			return true
		}
	}

	return false
}
