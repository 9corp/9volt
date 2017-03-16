// An abstraction layer for accessing data in etcd
package dal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/coreos/etcd/client"
)

//go:generate counterfeiter -o ../fakes/dalfakes/fake_dal.go dal.go IDal

type IDal interface {
	Get(string, *GetOptions) (map[string]string, error)
	Set(string, string, *SetOptions) error
	Delete(string, bool) error
	Refresh(string, int) error
	KeyExists(string) (bool, bool, error)
	IsKeyNotFound(error) bool
	CreateDirectorState(string) error
	UpdateDirectorState(string, string, bool) error
	NewWatcher(string, bool) client.Watcher
	GetClusterMembers() ([]string, error)
	GetCheckKeys() ([]string, error)
	GetCheckKeysWithMemberTag() (map[string]string, error)
	CreateCheckReference(string, string) error
	ClearCheckReference(string, string) error
	ClearCheckReferences(string) error
	FetchAllMemberRefs() (map[string]string, []string, error)
	FetchCheckStats() (map[string]*MemberStat, error)
	FetchAlerterConfig(string) (string, error)
	FetchState() ([]byte, error)
	FetchStateWithTags([]string) ([]byte, error)
	UpdateCheckState(bool, string) error
	GetClusterStats() (*ClusterStats, error)
	FetchEvents([]string) ([]byte, error)
	GetClusterMembersWithTags() (map[string][]string, error)
	GetClusterMemberTags(string) ([]string, error)
	GetCheckMemberTag(string) (string, error)
	PushConfigs(string, map[string][]byte) (int, int, error)
	PushFullConfigs(*FullConfigs) (*CfgUtilPushStats, []string)
}

type GetOptions struct {
	Recurse  bool
	NoPrefix bool

	// GetOptions modifies this internal prefix state based on call
	prefix string
}

//go:generate counterfeiter -o ../fakes/etcdclientfakes/fake_keysapi.go ../vendor/github.com/coreos/etcd/client/keys.go KeysAPI
//go:generate perl -pi -e s/github.com\/9corp\/9volt\/vendor\///g ../fakes/etcdclientfakes/fake_keysapi.go

type Dal struct {
	Client  client.Client
	KeysAPI client.KeysAPI
	Members []string
	Prefix  string
	Replace bool
	Dryrun  bool
	Nosync  bool
}

// Helper struct for FetchCheckStats()
type MemberStat struct {
	NumChecks int
	Tags      []string
}

type FullConfigs struct {
	AlerterConfigs map[string][]byte // alerter name : json blob
	MonitorConfigs map[string][]byte // monitor name : json blob
}

func New(prefix string, members []string, replace, dryrun, nosync bool) (*Dal, error) {
	log.Debugf("Connecting to etcd cluster with members: %v", members) //needs to be before any errs

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

type SetOptions struct {
	// If SetOptions.Dir=true then value is ignored.
	Dir       bool
	TTLSec    int
	PrevExist string

	// Create parents will recursively create parent directories as needed.
	CreateParents bool

	// To be used with create parents.
	// If depth > 0 it will only create depth number of parents.
	// If depth < 0 it will try to create as many parents as necessary.
	// If depth == 0 it will not create any parents. This behaves the same as Set().
	Depth int
}

// A wrapper for setting a new key
func (d *Dal) Set(key, value string, opt *SetOptions) error {
	if key[0] != '/' {
		key = "/" + key
	}

	// enforce this just in case
	if !opt.CreateParents {
		opt.Depth = 0
	}

	err := d.setAndCreateParents(
		d.Prefix+key,
		value,
		opt.Dir,
		opt.Depth,
		time.Duration(opt.TTLSec)*time.Second,
		client.PrevExistType(opt.PrevExist),
	)

	return err
}

// Helper method for set to recursively create its parents if they do not exist.
// It will create up to `depth` number of parents.
func (d *Dal) setAndCreateParents(
	key, value string, dir bool, depth int, ttl time.Duration, pExist client.PrevExistType) error {

	_, err := d.KeysAPI.Set(
		context.Background(),
		key,
		value,
		&client.SetOptions{
			Dir:       dir,
			TTL:       ttl,
			PrevExist: pExist,
		},
	)

	if err != nil {
		// is an etcd error
		etcdErr, ok := err.(client.Error)
		if !ok {
			return err
		}

		// parent creation is enabled, and error is a key not found
		if depth != 0 && etcdErr.Code == client.ErrorCodeKeyNotFound {
			// recursively create its parent first
			parent := key[:strings.LastIndex(key, "/")]
			if err := d.setAndCreateParents(parent, "", true, depth-1, ttl, client.PrevNoExist); err != nil {
				return err
			}

			// now try to set it again
			if _, err := d.KeysAPI.Set(
				context.Background(),
				key,
				value,
				&client.SetOptions{
					Dir:       dir,
					TTL:       ttl,
					PrevExist: pExist,
				},
			); err != nil {
				// give up if that didnt work
				return err
			}

			return nil
		}

		// depth == 0 or got etcd error other than key not found
		return err
	}

	return nil
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

// Get wrapper; either returns the key contents or error; accepts *GetOptions for
// specifying whether the method should recurse and/or use the default prefix.
//
// By default, passing a nil for Options will NOT recurse and use the default
// prefix of `d.Prefix`. Passing in a `GetOptions{NoPrefix: true}` will cause
// GET to not use ANY prefix (assuming key name includes full path).
//
// Returns a map[keyname]value; if dir is empty, return an empty map.
func (d *Dal) Get(key string, getOptions *GetOptions) (map[string]string, error) {
	// if given no options, instantiate default GetOptions
	if getOptions == nil {
		getOptions = &GetOptions{}
	}

	// If we ARE supposed to use a prefix, use the default one
	if !getOptions.NoPrefix {
		getOptions.prefix = d.Prefix
	}

	resp, err := d.KeysAPI.Get(context.Background(), getOptions.prefix+"/"+key, nil)
	if err != nil {
		return nil, err
	}

	values := make(map[string]string, 0)

	// If recurse is set, verify the key is a dir
	if getOptions.Recurse {
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
func (d *Dal) FetchAllMemberRefs() (map[string]string, []string, error) {
	// Fetch all cluster members
	memberIDs, err := d.GetClusterMembers()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to fetch cluster members: %v", err.Error())
	}

	memberRefs := make(map[string]string, 0)
	freeMembers := make([]string, 0)

	// Recursively fetch every member ref and build our memberRefs structure
	for _, memberID := range memberIDs {
		refs, err := d.Get(fmt.Sprintf("/cluster/members/%v/config", memberID), &GetOptions{
			Recurse: true,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("Problem fetching refs for '%v': %v", memberID, err.Error())
		}

		if len(refs) == 0 {
			freeMembers = append(freeMembers, memberID)
			continue
		}

		for _, v := range refs {
			memberRefs[v] = memberID
		}
	}

	return memberRefs, freeMembers, nil
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
	// A recursive delete removes all child entries AND the dir itself; this is
	// a no-go for us, so we first perform a recursive fetch and then remove
	// each individual entry. See: https://github.com/coreos/etcd/issues/2385
	data, err := d.Get("/cluster/members/"+memberID+"/config/", &GetOptions{
		Recurse: true,
	})
	if err != nil {
		if !client.IsKeyNotFound(err) {
			return err
		}
	}

	for key, _ := range data {
		_, err := d.KeysAPI.Delete(
			context.Background(),
			key,
			&client.DeleteOptions{
				Recursive: false,
			},
		)

		if err != nil {
			return fmt.Errorf("Unable to delete '%v' during ClearCheckReferences: %v", key, err)
		}
	}

	return nil
}

// Get slice of all member id's under /cluster/members/*
func (d *Dal) GetClusterMembers() ([]string, error) {
	data, err := d.Get("cluster/members/", &GetOptions{
		Recurse: true,
	})

	if err != nil {
		return nil, err
	}

	members := make([]string, 0)

	for k := range data {
		members = append(members, path.Base(k))
	}

	return members, nil
}

// Keys == member id, value == slice of tags
func (d *Dal) GetClusterMembersWithTags() (map[string][]string, error) {
	members, err := d.GetClusterMembers()
	if err != nil {
		return nil, fmt.Errorf("Unable to perform initial cluster member fetch: %v", err)
	}

	memberMap := make(map[string][]string, 0)

	for _, v := range members {
		key := "cluster/members/" + v + "/status"
		status, err := d.Get(key, nil)
		if err != nil {
			return nil, fmt.Errorf("Unable to fetch member '%v' status: %v", v, err)
		}

		if _, ok := status[key]; !ok {
			return nil, fmt.Errorf("Could not lookup member '%v' status in return map", v)
		}

		tags, err := d.parseTags(status[key])
		if err != nil {
			return nil, fmt.Errorf("Unable to fetch tags from member '%v' status: %v", v, err)
		}

		memberMap[v] = tags
	}

	return memberMap, nil
}

// Helper for parsing 'member-tag' from monitor config JSON payload
func (d *Dal) parseMemberTag(data string) (string, error) {
	var tmpTag struct {
		MemberTag string `json:"member-tag"`
	}

	if err := json.Unmarshal([]byte(data), &tmpTag); err != nil {
		return "", fmt.Errorf("Unable to complete member-tag unmarshal: %v", err)
	}

	return tmpTag.MemberTag, nil
}

// Helper for parsing 'Tags' or 'tags' array from cluster member JSON payload
func (d *Dal) parseTags(data string) ([]string, error) {
	var tmpTags struct {
		Tags []string
	}

	if err := json.Unmarshal([]byte(data), &tmpTags); err != nil {
		return nil, fmt.Errorf("Unable to complete tag unmarshal: %v", err)
	}

	return tmpTags.Tags, nil
}

// Fetch how many checks each cluster member has
func (d *Dal) FetchCheckStats() (map[string]*MemberStat, error) {
	memberRefs, freeMembers, err := d.FetchAllMemberRefs()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch memberRefs for FetchCheckStats(): %v", err.Error())
	}

	memberTags, err := d.GetClusterMembersWithTags()
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch member tags for FetchCheckStats(): %v", err)
	}

	checkStats := make(map[string]*MemberStat)

	// Let's record the members that have checks already assigned to them
	for _, memberID := range memberRefs {
		if _, ok := checkStats[memberID]; !ok {
			checkStats[memberID] = &MemberStat{}
		}

		checkStats[memberID].NumChecks = checkStats[memberID].NumChecks + 1
	}

	// Let's record the members that have no checks assigned to them
	for _, memberID := range freeMembers {
		if _, ok := checkStats[memberID]; !ok {
			checkStats[memberID] = &MemberStat{}
		}

		checkStats[memberID].NumChecks = 0
	}

	// Let's assign tags to each
	for memberID, tags := range memberTags {
		if _, ok := checkStats[memberID]; !ok {
			return nil, errors.New("FetchCheckStats potential bug - memberTags contains a member " +
				"that does not exist in memberRefs or freeMembers")
		}

		checkStats[memberID].Tags = tags
	}

	return checkStats, nil
}

// Get a slice of all check keys in etcd (under /monitor/*)
func (d *Dal) GetCheckKeys() ([]string, error) {
	data, err := d.Get("monitor/", &GetOptions{
		Recurse: true,
	})

	if err != nil {
		return nil, err
	}

	checkKeys := make([]string, 0)

	for k := range data {
		checkKeys = append(checkKeys, k)
	}

	return checkKeys, nil
}

func (d *Dal) GetCheckMemberTag(checkKey string) (string, error) {
	data, err := d.Get(checkKey, &GetOptions{
		NoPrefix: true,
	})
	if err != nil {
		return "", fmt.Errorf("Unable to fetch member-tag for check '%v': %v", checkKey, err)
	}

	memberTag, err := d.parseMemberTag(data[checkKey])
	if err != nil {
		return "", fmt.Errorf("Unable to parse member-tag for check '%v': %v", checkKey, err)
	}

	return memberTag, nil
}

// Return check keys along with tags (if any); map k = check key name, v = member tag (if any)
func (d *Dal) GetCheckKeysWithMemberTag() (map[string]string, error) {
	data, err := d.Get("monitor/", &GetOptions{
		Recurse: true,
	})

	if err != nil {
		return nil, err
	}

	checkKeys := make(map[string]string, 0)

	for k, v := range data {
		// TODO: Should probably bubble-up error to event stream (and not critically fail)
		memberTag, err := d.parseMemberTag(v)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse member-tag for check config '%v': %v", k, err)
		}

		checkKeys[k] = memberTag
	}

	return checkKeys, nil
}

// Get tags for a single cluster member
func (d *Dal) GetClusterMemberTags(memberID string) ([]string, error) {
	fullKey := "/cluster/members/" + memberID + "/status"
	data, err := d.Get(fullKey, nil)
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch cluster member status for '%v': %v", memberID, err)
	}

	if _, ok := data[fullKey]; !ok {
		return nil, fmt.Errorf("Returned data set does not contain our expected key '%v'", fullKey)
	}

	tags, err := d.parseTags(data[fullKey])
	if err != nil {
		return nil, fmt.Errorf("Unable to parse member tags for '%v': %v", memberID, err)
	}

	return tags, nil
}

// Fetch a specific alerter config by its key name
func (d *Dal) FetchAlerterConfig(alertKey string) (string, error) {
	data, err := d.Get("alerter/"+alertKey, nil)
	if err != nil {
		return "", err
	}

	return data["alerter/"+alertKey], nil
}

// wrapper for etcd client's KeyNotFound error
func (d *Dal) IsKeyNotFound(err error) bool {
	return client.IsKeyNotFound(err)
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
