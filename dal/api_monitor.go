package dal

import (
	"encoding/json"
	"fmt"
)

func (d *Dal) UpdateCheckState(state bool, checkName string) error {
	fullPath := "monitor/" + checkName

	// Fetch check configuration
	data, err := d.Get(fullPath, nil)
	if err != nil {
		return err
	}

	var tmp map[string]interface{}

	if err := json.Unmarshal([]byte(data[fullPath]), &tmp); err != nil {
		return fmt.Errorf("Unable to perform unmarshal on '%v' config: %v", checkName, err)
	}

	// update check configuration
	tmp["disable"] = state

	newData, err := json.Marshal(tmp)
	if err != nil {
		return fmt.Errorf("Unable to marshal final config for '%v': %v", checkName, err)
	}

	// push updated config to etcd
	if err := d.Set(fullPath, string(newData), &SetOptions{Dir: false, TTLSec: 0, PrevExist: ""}); err != nil {
		return fmt.Errorf("Unable to update config '%v' in etcd: %v", checkName, err)
	}

	return nil
}
