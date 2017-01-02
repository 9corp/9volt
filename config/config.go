package config

import (
	"encoding/json"
	"fmt"

	// log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/event"
	"github.com/9corp/9volt/util"
)

type Config struct {
	MemberID      string
	ListenAddress string
	EtcdPrefix    string
	EtcdMembers   []string
	DalClient     dal.IDal
	EQClient      *event.Client

	serverConfig
}

type serverConfig struct {
	StateDumpInterval util.CustomDuration
	HeartbeatInterval util.CustomDuration
	HeartbeatTimeout  util.CustomDuration
}

// Pass in the dal client in order to facilitate better/easier testing story
func New(memberID, listenAddress, etcdPrefix string, etcdMembers []string, dalClient dal.IDal, eqClient *event.Client) *Config {
	cfg := &Config{
		ListenAddress: listenAddress,
		EtcdPrefix:    etcdPrefix,
		EtcdMembers:   etcdMembers,
		DalClient:     dalClient,
		EQClient:      eqClient,
		MemberID:      memberID,
	}

	return cfg
}

func (c *Config) ValidateDirs() []string {
	dirs := []string{"cluster", "cluster/members", "monitor", "alerter"}

	var errorList []string

	for _, d := range dirs {
		exists, isDir, err := c.DalClient.KeyExists(d)
		if err != nil {
			errorList = append(errorList, fmt.Sprintf("dal returned error when validating key '%v' in etcd: %v", d, err.Error()))
			continue
		}

		if !exists {
			errorList = append(errorList, fmt.Sprintf("required key '%v' does not exist", d))
			continue
		}

		if !isDir {
			errorList = append(errorList, fmt.Sprintf("required key '%v' exists, but is not of dir type", d))
			continue
		}
	}

	return errorList
}

func (c *Config) Load() error {
	exists, isDir, err := c.DalClient.KeyExists("config")
	if err != nil {
		return fmt.Errorf("dal error verifying 'config' key: %v", err.Error())
	}

	if !exists {
		return fmt.Errorf("'config' does not appear to exist in etcd")
	}

	if isDir {
		return fmt.Errorf("'config' exists but is a dir")
	}

	values, err := c.DalClient.Get("config", nil)
	if err != nil {
		return err
	}

	if _, ok := values["config"]; !ok {
		return fmt.Errorf("'config' missing in return data set (bug?)")
	}

	if err := c.load(values["config"]); err != nil {
		return err
	}

	return nil
}

func (c *Config) load(config string) error {
	var sc serverConfig

	if err := json.Unmarshal([]byte(config), &sc); err != nil {
		return fmt.Errorf("Unable to unmarshal server config: %v", err.Error())
	}

	if err := c.validate(&sc); err != nil {
		return fmt.Errorf("Unable to validate server config: %v", err.Error())
	}

	c.serverConfig = sc

	return nil
}

func (c *Config) validate(sc *serverConfig) error {
	if sc.HeartbeatInterval == 0 {
		return fmt.Errorf("'HeartbeatInterval' cannot be 0")
	}

	if sc.HeartbeatTimeout == 0 {
		return fmt.Errorf("'HeartbeatTimeout' cannot be 0")
	}

	if sc.StateDumpInterval == 0 {
		return fmt.Errorf("'StateDumpInterval' cannot be 0")
	}

	return nil
}
