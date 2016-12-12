package dal

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"

	"github.com/9corp/9volt/9volt-cfg/config"
)

type Dal struct {
	Client  client.Client
	KeysAPI client.KeysAPI
	Members []string
	Prefix  string
	Replace bool
	Dryrun  bool
	Nosync  bool
}

type PushStats struct {
	MonitorAdded   int
	AlerterAdded   int
	MonitorSkipped int
	AlerterSkipped int
	MonitorRemoved int
	AlerterRemoved int
}

func New(members []string, prefix string, replace, dryrun, nosync bool) (*Dal, error) {
	etcdClient, err := client.New(client.Config{
		Endpoints: members,
		Transport: client.DefaultTransport,
	})

	if err != nil {
		return nil, err
	}

	return &Dal{
		Client:  etcdClient,
		KeysAPI: client.NewKeysAPI(etcdClient),
		Members: members,
		Prefix:  prefix,
		Replace: replace,
		Dryrun:  dryrun,
		Nosync:  nosync,
	}, nil
}

func (d *Dal) Push(fullConfigs *config.FullConfigs) (*PushStats, []string) {
	errorList := make([]string, 0)

	mAdded, mSkipped, err := d.push("monitor", fullConfigs.MonitorConfigs)
	if err != nil {
		errorList = append(errorList, err.Error())
		log.Errorf("Unable to complete monitor config push: %v", err.Error())
	}

	aAdded, aSkipped, err := d.push("alerter", fullConfigs.AlerterConfigs)
	if err != nil {
		errorList = append(errorList, err.Error())
		log.Errorf("Unable to complete alerter config push: %v", err.Error())
	}

	pushStats := &PushStats{
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
func (d *Dal) sync(fullConfigs *config.FullConfigs) (int, int, error) {
	count := map[string]int{"monitor": 0, "alerter": 0}

	etcdKeys, err := d.getEtcdKeys()
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to fetch all keys from etcd: %v", err.Error())
	}

	// get all of our keys
	configKeys := make(map[string][]string, 0)

	configKeys["alerter"] = d.getMapKeys(fullConfigs.AlerterConfigs)
	configKeys["monitor"] = d.getMapKeys(fullConfigs.MonitorConfigs)

	for etcdConfigType, etcdKeyNames := range etcdKeys {
		// let's roll through the keys in etcd
		for _, etcdKeyName := range etcdKeyNames {
			if !d.stringSliceContains(configKeys[etcdConfigType], etcdKeyName) {

				if !d.Nosync {
					removeKey := fmt.Sprintf("%v/%v", etcdConfigType, etcdKeyName)
					log.Debugf("Sync: Removing orphaned '%v' config from etcd", removeKey)

					if err := d.remove(removeKey); err != nil {
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

// Fetch all alerter and monitor keys, return as map containing config type and
// slice of keys
func (d *Dal) getEtcdKeys() (map[string][]string, error) {
	keyMap := map[string][]string{
		"alerter": make([]string, 0),
		"monitor": make([]string, 0),
	}

	for k := range keyMap {
		fullPath := "/" + d.Prefix + "/" + k + "/"

		resp, err := d.KeysAPI.Get(context.Background(), fullPath, nil)
		if err != nil {
			return nil, err
		}

		if !resp.Node.Dir {
			return nil, fmt.Errorf("Etcd problem: %v is not a dir!", fullPath)
		}

		for _, etcdKey := range resp.Node.Nodes {
			keyMap[k] = append(keyMap[k], filepath.Base(etcdKey.Key))
		}
	}

	return keyMap, nil
}

// Helper for determining if a string slice contains given string
func (d *Dal) stringSliceContains(stringSlice []string, data string) bool {
	for _, v := range stringSlice {
		if v == data {
			return true
		}
	}

	return false
}

// Helper for fetching keys in a map
func (d *Dal) getMapKeys(inputMap map[string][]byte) []string {
	keys := make([]string, len(inputMap))

	for k := range inputMap {
		keys = append(keys, k)
	}

	return keys
}

// Wrapper for comparing existing value in etcd + (potentially) pushing value to etcd.
// If d.Replace is set, value in etcd will be replaced even if it's found to match
func (d *Dal) push(configType string, configs map[string][]byte) (int, int, error) {
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

// Remove a given key from etcd
func (d *Dal) remove(key string) error {
	_, err := d.KeysAPI.Delete(
		context.Background(),
		d.Prefix+"/"+key,
		&client.DeleteOptions{
			Recursive: false,
		},
	)

	return err
}

// Check if given key exists in etcd, if it does, determine if its value
// matches new value by performing a reflect.DeepEqual().
func (d *Dal) compare(fullPath string, data []byte) (bool, error) {
	resp, err := d.KeysAPI.Get(context.Background(), fullPath, nil)
	if err != nil {
		if client.IsKeyNotFound(err) {
			return false, nil
		}

		return false, err
	}

	// Unmarshal and compare both entries
	var etcdEntry interface{}
	var newEntry interface{}

	if err := json.Unmarshal([]byte(resp.Node.Value), &etcdEntry); err != nil {
		return false, fmt.Errorf("Unable to unmarshal existing entry in etcd '%v': %v", fullPath, err.Error())
	}

	if err := json.Unmarshal(data, &newEntry); err != nil {
		return false, fmt.Errorf("Unable to unmarshal existing entry in etcd '%v': %v", fullPath, err.Error())
	}

	return reflect.DeepEqual(etcdEntry, newEntry), nil
}
