package config

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"

	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/util"
)

type Config struct {
	ListenAddress string
	EtcdPrefix    string
	EtcdMembers   []string
	serverConfig
}

type serverConfig struct {
	HeartbeatInterval util.CustomDuration
	HeartbeatTimeout  util.CustomDuration
}

func New(listenAddress, etcdPrefix string, etcdMembers []string) *Config {
	cfg := &Config{
		ListenAddress: listenAddress,
		EtcdPrefix:    etcdPrefix,
		EtcdMembers:   etcdMembers,
	}

	return cfg
}

func (c *Config) Load() error {
	dalClient, err := dal.New(c.EtcdPrefix, c.EtcdMembers)
	if err != nil {
		return err
	}

	values, notFound, err := dalClient.Get("config", false)
	if err != nil {
		return err
	}

	if notFound {
		return fmt.Errorf("'config' does not appear to exist in etcd")
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
	log.Warningf("serverConfig contents: %v", sc)
	return nil
}
