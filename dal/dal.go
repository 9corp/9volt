// An abstraction layer for accessing data in etcd
package dal

import (
	"fmt"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

type IDal interface {
	Get(string, bool) (map[string]string, bool, error)
	KeyExists(string) (bool, bool, error)
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

// Get wrapper; either returns the key contents, 'key not found' bool or error
func (d *Dal) Get(key string, recurse bool) (map[string]string, bool, error) {
	resp, err := d.KeysAPI.Get(context.Background(), d.Prefix+"/"+key, nil)

	if err != nil {
		if client.IsKeyNotFound(err) {
			return nil, true, nil
		}

		return nil, false, err
	}

	values := make(map[string]string, 0)

	// If recurse is set, verify the key is a dir
	if recurse {
		if !resp.Node.Dir {
			return nil, false, fmt.Errorf("Recurse is enabled, but '%v' is not a dir", key)
		}

		// Dir is empty; return empty map
		if resp.Node.Nodes.Len() == 0 {
			return values, false, nil
		}

		for _, val := range resp.Node.Nodes {
			values[val.Key] = val.Value
		}
	} else {
		values[key] = resp.Node.Value
	}

	return values, false, nil
}
