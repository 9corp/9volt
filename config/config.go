package config

import ()

type Config struct {
	ListenAddress string
	EtcdPrefix    string
	EtcdMembers   []string
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
	return nil
}
