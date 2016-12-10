package dal

import (
	"context"
	"encoding/json"
	"fmt"
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
}

type PushStats struct {
	MonitorAdded   int
	AlerterAdded   int
	MonitorSkipped int
	AlerterSkipped int
}

func New(members []string, prefix string, replace, dryrun bool) (*Dal, error) {
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

	return &PushStats{
		MonitorAdded:   mAdded,
		AlerterAdded:   aAdded,
		MonitorSkipped: mSkipped,
		AlerterSkipped: aSkipped,
	}, errorList
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
