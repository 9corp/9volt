package dal

import (
	"encoding/json"
	"fmt"
	"path"
	// log "github.com/Sirupsen/logrus"
)

func (d *Dal) FetchEvents() ([]byte, error) {
	eventData, err := d.Get("event", &GetOptions{
		Recurse: true,
	})

	if err != nil {
		return nil, err
	}

	combinedData := make(map[string]*json.RawMessage, len(eventData))

	for k, v := range eventData {
		event := json.RawMessage(v)
		combinedData[path.Base(k)] = &event
	}

	eventBlob, err := json.Marshal(combinedData)
	if err != nil {
		return nil, fmt.Errorf("Unable to re-marshal event data: %v", err)
	}

	return eventBlob, nil
}

func (d *Dal) FetchEventsWithTypes(types []string) ([]byte, error) {
	return nil, nil
}
