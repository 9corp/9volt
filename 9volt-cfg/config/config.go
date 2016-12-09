package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Dir string
}

func New(dir string) (*Config, error) {
	// Verify if the dir exists (or is a dir)
	if err := validateDir(dir); err != nil {
		return nil, err
	}

	return &Config{
		Dir: dir,
	}, nil
}

func (c *Config) Fetch() ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(c.Dir, func(path string, info os.FileInfo, err error) error {
		fullPath := fmt.Sprintf("%v/%v", c.Dir, path)

		if !strings.HasSuffix(path, ".yaml") {
			log.Debugf("Skipping file %v", fullPath)
			return nil
		}

		// read file
		data, err := ioutil.ReadFile(fullPath)
		if err != nil {
			log.Warningf("Skipping unreadable file %v: %v", fullPath, err.Error())
			return nil
		}

		var test map[string]interface{}

		// try to unmarshal it to see if it's yaml or not
		if err := yaml.Unmarshal(data, &test); err != nil {
			log.Warningf("Skipping non-YAML file %v: %v", fullPath, err.Error())
			return nil
		}

		files = append(files, fullPath)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (c *Config) Parse(files []string) ([]string, error) {
	return []string{}, nil
}

func (c *Config) Validate(configs []string) error {
	return nil
}

func validateDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("No such file or directory '%v'", dir)
		} else {
			return err
		}
	}

	if !info.IsDir() {
		return fmt.Errorf("'%v' does not appear to be a directory", dir)
	}

	return nil
}
