package config

import (
// log "github.com/Sirupsen/logrus"
)

type Config struct {
	Dir string
}

func New(dir string) (*Config, error) {
	// Verify if the dir exists (or is a dir)

	return &Config{
		Dir: dir,
	}, nil
}

func (c *Config) Fetch() ([]string, error) {
	return []string{}, nil
}

func (c *Config) Parse(files []string) ([]string, error) {
	return []string{}, nil
}

func (c *Config) Validate(configs []string) error {
	return nil
}
