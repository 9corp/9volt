package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
)

type Config struct {
	Dir string
}

type FullConfigs struct {
	AlerterConfigs map[string][]byte // alerter name : json blob
	MonitorConfigs map[string][]byte // monitor name : json blob
}

type YAMLFileBlob map[string]map[string]interface{}

// Instantiate a new Config client; ensure the given directory exists (and is a directory)
func New(dir string) (*Config, error) {
	// Verify if the dir exists (or is a dir)
	if err := validateDir(dir); err != nil {
		return nil, err
	}

	return &Config{
		Dir: dir,
	}, nil
}

// Recursively walk through 'dir', find any .yaml files and verify that they are
// in fact files containing valid YAML. If they do not contain valid YAML, skip
// file and display warning.
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

// Roll through the list of YAML files, parsing each for all 'alerter' and 'monitor'
// sections; parse (and validate) each section, convert it from YAML -> JSON
// and construct a response with MonitorConfigs and AlerterConfigs.
//
// Structure for MonitorConfigs and AlerterConfigs is a map where the key is the
// keyname for the config and the vaue is the JSON blob as a byte slice.
func (c *Config) Parse(files []string) (*FullConfigs, error) {
	fullConfigs := &FullConfigs{
		AlerterConfigs: make(map[string][]byte, 0),
		MonitorConfigs: make(map[string][]byte, 0),
	}

	for _, file := range files {
		// read the file
		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Warningf("Unable to read file %v: %v", file, err.Error())
			continue
		}

		configTypes, yamlData, err := c.containsConfigs(data)
		if err != nil {
			log.Warningf("Unable to determine if '%v' contains configs: %v", err.Error())
			continue
		}

		// Roll through monitor and/or alerter configs
		for _, configType := range configTypes {
			// validate the config first
			if err := c.validate(configType, yamlData[configType]); err != nil {
				log.Warningf("Unable to validate %v configs in %v: %v", configType, file, err.Error())
				continue
			}

			// convert the configs
			jsonConfigs, err := c.convertToJSON(yamlData[configType])
			if err != nil {
				log.Warningf("Unable to convert %v configs in %v to JSON: %v", configType, file, err.Error())
				continue
			}

			// save the configs
			for k, v := range jsonConfigs {
				switch configType {
				case "alerter":
					if _, ok := fullConfigs.AlerterConfigs[k]; ok {
						log.Warningf("Skipping dupe entry for alerter config '%v' detected in '%v'!", k, file)
						continue
					}

					fullConfigs.AlerterConfigs[k] = v
				case "monitor":
					if _, ok := fullConfigs.MonitorConfigs[k]; ok {
						log.Warningf("Skipping dupe entry for alerter config '%v' detected in '%v'!", k, file)
						continue
					}

					fullConfigs.MonitorConfigs[k] = v
				default:
					log.Errorf("Unexpected behavior while saving configs from %v", file)
				}
			}
		}
	}

	return fullConfigs, nil
}

// Convert YAML blob -> JSON *after* validation
func (c *Config) convertToJSON(data map[string]interface{}) (map[string][]byte, error) {
	converted := make(map[string][]byte, 0)

	for name, yamlBlob := range data {
		jsonBlob, err := json.Marshal(yamlBlob)
		if err != nil {
			return nil, fmt.Errorf("Unable to marshal '%v' YAML portion to JSON: %v", name, err.Error())
		}

		converted[name] = jsonBlob
	}

	return converted, nil
}

// Validate given type config
// data == configKey : yaml data; monitor or alerter type determined by 'configType'
func (c *Config) validate(configType string, data map[string]interface{}) error {
	// TODO: perform validation according to the type of configType we got
	return nil
}

func (c *Config) containsConfigs(data []byte) ([]string, YAMLFileBlob, error) {
	// try to unmarshal entire file and verify if it contains 'alerter' or 'monitor'
	var yamlData YAMLFileBlob

	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return nil, nil, err
	}

	configTypes := []string{}

	if _, ok := yamlData["alerter"]; ok {
		configTypes = append(configTypes, "alerter")
	}

	if _, ok := yamlData["monitor"]; ok {
		configTypes = append(configTypes, "monitor")
	}

	return configTypes, yamlData, nil
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
