package dal

import (
	"context"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"

	"github.com/9corp/9volt/util"
)

type CfgUtilPushStats struct {
	MonitorAdded   int
	AlerterAdded   int
	MonitorSkipped int
	AlerterSkipped int
	MonitorRemoved int
	AlerterRemoved int
}

// Wrapper for comparing existing value in etcd + (potentially) pushing value to etcd.
// If d.Replace is set, value in etcd will be replaced even if it's found to match
func (d *Dal) PushConfigs(configType string, configs map[string][]byte) (int, int, error) {
	added := 0
	skipped := 0

	for k, v := range configs {
		fullPath := d.Prefix + "/" + configType + "/" + k

		// see if key exists and matches content
		match, err := d.compare(fullPath, v)
		if err != nil {
			log.Errorf("Experienced an error during etcd compare for %v: %v", fullPath, err.Error())
			return added, skipped, err
		}

		// Skip if an identical config exists and forced replace is not enabled
		if match && !d.Replace {
			skipped++

			log.Debugf("Skipping push for matching %v config '%v' in etcd", configType, k)
			continue
		}

		if !d.Dryrun {
			if err := d.pushConfig(fullPath, v); err != nil {
				log.Errorf("Experienced an error during etcd push for %v: %v", fullPath, err.Error())
				return added, skipped, err
			}
		}

		added++
	}

	return added, skipped, nil
}

// Wrapper for pushing monitor and alerter configs + optionally syncing data
func (d *Dal) PushFullConfigs(fullConfigs *FullConfigs) (*CfgUtilPushStats, []string) {
	errorList := make([]string, 0)

	mAdded, mSkipped, err := d.PushConfigs("monitor", fullConfigs.MonitorConfigs)
	if err != nil {
		errorList = append(errorList, err.Error())
		log.Errorf("Unable to complete monitor config push: %v", err.Error())
	}

	aAdded, aSkipped, err := d.PushConfigs("alerter", fullConfigs.AlerterConfigs)
	if err != nil {
		errorList = append(errorList, err.Error())
		log.Errorf("Unable to complete alerter config push: %v", err.Error())
	}

	pushStats := &CfgUtilPushStats{
		MonitorAdded:   mAdded,
		AlerterAdded:   aAdded,
		MonitorSkipped: mSkipped,
		AlerterSkipped: aSkipped,
	}

	// If syncing is enabled (default), remove any configs from etcd that do not
	// have a corresponding fullConfigs entry
	if !d.Nosync {
		mRemoved, aRemoved, err := d.sync(fullConfigs)

		if err != nil {
			log.Errorf("Unable to complete sync: %v", err.Error())
		} else {
			pushStats.MonitorRemoved = mRemoved
			pushStats.AlerterRemoved = aRemoved
		}
	}

	return pushStats, errorList
}

// Remove any configs from etcd that are not defined in fullConfigs
func (d *Dal) sync(fullConfigs *FullConfigs) (int, int, error) {
	count := map[string]int{"monitor": 0, "alerter": 0}

	etcdKeys, err := d.getEtcdKeys()
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to fetch all keys from etcd: %v", err.Error())
	}

	// get all of our keys
	configKeys := make(map[string][]string, 0)

	configKeys["alerter"] = util.GetMapKeys(fullConfigs.AlerterConfigs)
	configKeys["monitor"] = util.GetMapKeys(fullConfigs.MonitorConfigs)

	for etcdConfigType, etcdKeyNames := range etcdKeys {
		// let's roll through the keys in etcd
		for _, etcdKeyName := range etcdKeyNames {
			if !util.StringSliceContains(configKeys[etcdConfigType], etcdKeyName) {

				if !d.Nosync {
					removeKey := fmt.Sprintf("%v/%v", etcdConfigType, etcdKeyName)
					log.Debugf("Sync: Removing orphaned '%v' config from etcd", removeKey)

					if err := d.Delete(removeKey, false); err != nil {
						log.Errorf("Unable to remove orphaned config '%v': %v", removeKey, err.Error())
						continue
					}
				}

				count[etcdConfigType]++

			}
		}
	}

	return count["monitor"], count["alerter"], nil
}

// Push data blob to given key in etcd; do not care about previous setting
func (d *Dal) pushConfig(fullPath string, data []byte) error {
	_, err := d.KeysAPI.Set(
		context.Background(),
		fullPath,
		string(data),
		&client.SetOptions{
			PrevExist: client.PrevIgnore,
		},
	)

	return err
}
