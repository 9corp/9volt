// An abstraction layer for accessing data in etcd
package dal

import (
	"encoding/base64"
	"fmt"
	"path"
	"time"

	// log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type IDal interface {
	Get(string, bool) (map[string]string, error)
	Set(string, string, bool, int, string) error
	Delete(string, bool) error
	Refresh(string, int) error
	KeyExists(string) (bool, bool, error)
	IsKeyNotFound(error) bool
	CreateDirectorState(string) error
	UpdateDirectorState(string, string, bool) error
	NewWatcher(string, bool) client.Watcher
	GetClusterMembers() ([]string, error)
	GetCheckKeys() ([]string, error)
	CreateCheckReference(string, string) error
	ClearCheckReference(string, string) error
	ClearCheckReferences(string) error
	FetchAllMemberRefs() (map[string]string, error)
	FetchCheckStats() (map[string]int, error)
}

type Dal struct {
	Client  client.Client
	KeysAPI client.KeysAPI
	Members []string
	Prefix  string
}

func New(prefix string, members []string) (IDal, error) {
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
	}, nil
}

// Check if a given key exists and whether it's a dir or not (or return error)
func (d *Dal) KeyExists(key string) (bool, bool, error) {
	resp, err := d.KeysAPI.Get(context.Background(), d.Prefix+"/"+key, nil)

	if err != nil {
		if client.IsKeyNotFound(err) {
			return false, false, nil
		}

		return false, false, err
	}

	return true, resp.Node.Dir, nil
}

// An unwieldy wrapper for setting a new key
func (d *Dal) Set(key, value string, dir bool, ttl int, prevExist string) error {
	existState := client.PrevExistType(prevExist)

	_, err := d.KeysAPI.Set(
		context.Background(),
		d.Prefix+"/"+key,
		value,
		&client.SetOptions{
			Dir:       dir,
			TTL:       time.Duration(ttl) * time.Second,
			PrevExist: existState,
		},
	)

	return err
}

// Set TTL for a given key
func (d *Dal) Refresh(key string, ttl int) error {
	_, err := d.KeysAPI.Set(
		context.Background(),
		d.Prefix+"/"+key,
		"",
		&client.SetOptions{
			Refresh:   true,
			PrevExist: client.PrevExist,
			TTL:       time.Duration(ttl) * time.Second,
		},
	)

	return err
}

// Get wrapper; either returns the key contents or error
func (d *Dal) Get(key string, recurse bool) (map[string]string, error) {
	resp, err := d.KeysAPI.Get(context.Background(), d.Prefix+"/"+key, nil)
	if err != nil {
		return nil, err
	}

	values := make(map[string]string, 0)

	// If recurse is set, verify the key is a dir
	if recurse {
		if !resp.Node.Dir {
			return nil, fmt.Errorf("Recurse is enabled, but '%v' is not a dir", key)
		}

		// Dir is empty; return empty map
		if resp.Node.Nodes.Len() == 0 {
			return values, nil
		}

		for _, val := range resp.Node.Nodes {
			values[val.Key] = val.Value
		}
	} else {
		values[key] = resp.Node.Value
	}

	return values, nil
}

// Create a check reference for a specific member under /cluster/members/*/config/*;
// check ref key is base64 encodded, value is set to the keyName
func (d *Dal) CreateCheckReference(memberID, keyName string) error {
	b64Key := base64.StdEncoding.EncodeToString([]byte(keyName))

	_, err := d.KeysAPI.Set(
		context.Background(),
		d.Prefix+"/cluster/members/"+memberID+"/config/"+b64Key,
		keyName,
		&client.SetOptions{
			PrevExist: client.PrevIgnore,
		},
	)

	return err
}

// Recursively fetch '/cluster/members/*/config/*', construct and return dataset
// that has 'map[check_key]memberID' structure
//
// TODO: This is not great; should utilize caching at some point
func (d *Dal) FetchAllMemberRefs() (map[string]string, error) {
	// Fetch all cluster members
	memberIDs, err := d.GetClusterMembers()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch cluster members: %v", err.Error())
	}

	memberRefs := make(map[string]string, 0)

	// Recursively fetch every member ref and build our memberRefs structure
	for _, memberID := range memberIDs {
		refs, err := d.Get(fmt.Sprintf("/cluster/members/%v/config", memberID), true)
		if err != nil {
			return nil, fmt.Errorf("Problem fetching refs for '%v': %v", memberID, err.Error())
		}

		for _, v := range refs {
			memberRefs[v] = memberID
		}
	}

	fmt.Printf("Contents of memberRefs: %v\n", memberRefs)

	return memberRefs, nil
}

// Create director state entry (expecting director state key to not exist)
func (d *Dal) CreateDirectorState(data string) error {
	_, err := d.KeysAPI.Set(
		context.Background(),
		d.Prefix+"/cluster/director",
		data,
		&client.SetOptions{
			PrevExist: client.PrevNoExist,
		},
	)

	return err
}

func (d *Dal) Delete(key string, recursive bool) error {
	_, err := d.KeysAPI.Delete(
		context.Background(),
		d.Prefix+"/"+key,
		&client.DeleteOptions{
			Recursive: recursive,
		},
	)

	return err
}

func (d *Dal) NewWatcher(key string, recursive bool) client.Watcher {
	return d.KeysAPI.Watcher(d.Prefix+"/"+key, &client.WatcherOptions{
		Recursive: recursive,
	})
}

// Update director state entry (expecting previous director state to match 'prevValue')
// (or force the update, ignoring prevValue)
func (d *Dal) UpdateDirectorState(data, prevValue string, force bool) error {
	setOptions := new(client.SetOptions)

	if !force {
		setOptions.PrevValue = prevValue
	}

	_, err := d.KeysAPI.Set(
		context.Background(),
		d.Prefix+"/cluster/director",
		data,
		setOptions,
	)

	return err
}

// Remove check reference for a given memberID + checkName
func (d *Dal) ClearCheckReference(memberID, keyName string) error {
	b64Key := base64.StdEncoding.EncodeToString([]byte(keyName))

	_, err := d.KeysAPI.Delete(
		context.Background(),
		d.Prefix+"/cluster/members/"+memberID+"/config/"+b64Key,
		nil,
	)

	return err
}

// Remove all key refs under individual member config dir
func (d *Dal) ClearCheckReferences(memberID string) error {
	_, err := d.KeysAPI.Delete(
		context.Background(),
		d.Prefix+"/cluster/members/"+memberID+"/config/",
		&client.DeleteOptions{
			Recursive: true,
			Dir:       false,
		},
	)

	// Prevent erroring on a 404
	if client.IsKeyNotFound(err) {
		return nil
	}

	return err
}

// Get slice of all member id's under /cluster/members/*
func (d *Dal) GetClusterMembers() ([]string, error) {
	data, err := d.Get("cluster/members/", true)
	if err != nil {
		return nil, err
	}

	members := make([]string, 0)

	for k, _ := range data {
		members = append(members, path.Base(k))
	}

	return members, nil
}

// Fetch how many checks each cluster member has
func (d *Dal) FetchCheckStats() (map[string]int, error) {
	memberRefs, err := d.FetchAllMemberRefs()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch memberRefs for FetchCheckStats(): %v", err.Error())
	}

	checkStats := make(map[string]int)

	for _, v := range memberRefs {
		checkStats[v] = checkStats[v] + 1
	}

	return checkStats, nil
}

// Get a slice of all check keys in etcd (under /monitor/*)
func (d *Dal) GetCheckKeys() ([]string, error) {
	data, err := d.Get("monitor/", true)
	if err != nil {
		return nil, err
	}

	checkKeys := make([]string, 0)

	for k, _ := range data {
		checkKeys = append(checkKeys, k)
	}

	return checkKeys, nil
}

// wrapper for etcd client's KeyNotFound error
func (d *Dal) IsKeyNotFound(err error) bool {
	return client.IsKeyNotFound(err)
}
